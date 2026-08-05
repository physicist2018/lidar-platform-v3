#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# auth.sh — регистрация и авторизация в identity-сервисе
#
# Использование:
#   ./scripts/auth.sh register <email> <password>          — регистрация
#   ./scripts/auth.sh login    <email> <password>          — вход, сохранение токенов
#   ./scripts/auth.sh full     <email> <password>          — регистрация + вход
#   ./scripts/auth.sh refresh                              — обновить пару токенов
#   ./scripts/auth.sh token                                — показать сохранённый access-токен
#   ./scripts/auth.sh refresh-token                        — показать сохранённый refresh-токен
#
# По умолчанию identity доступен на http://localhost:8090 (HTTP_ADDR из run_identity.sh).
# Можно переопределить через переменную IDENTITY_URL:
#   IDENTITY_URL=http://localhost:8090 ./scripts/auth.sh login user@ex.com pass
# ──────────────────────────────────────────────────────────────────────────────

IDENTITY_URL="${IDENTITY_URL:-https://localhost}"
TOKEN_FILE="${TOKEN_FILE:-/tmp/lidar_token.txt}"
REFRESH_TOKEN_FILE="${REFRESH_TOKEN_FILE:-/tmp/lidar_refresh_token.txt}"

# ==============================================================================
# Регистрация
# ==============================================================================
register() {
    local email="$1"
    local password="$2"

    echo "==> Registering user: $email"
    response=$(curl -k -s -w "\n%{http_code}" -X POST "$IDENTITY_URL/register" \
        -H "Content-Type: application/json" \
        -d "$(cat <<EOF
{"email": "$email", "password": "$password"}
EOF
)")

    body=$(echo "$response" | sed '$d')
    code=$(echo "$response" | tail -1)

    case "$code" in
        201)
            echo "    ✓ User registered"
            echo "    └─ Verification email sent to $email"
            echo ""
            echo "    Чтобы подтвердить учётную запись, выполните:"
            echo "        docker-compose exec -T db psql -U postgres -d main_db -c \\"
            echo "          \"UPDATE identity.users SET status='active' WHERE email='$email';\" 2>/dev/null ||"
            echo "        psql \"\$DATABASE_URL\" -c \"UPDATE identity.users SET status='active' WHERE email='$email';\""
            echo ""
            echo "    Или найдите токен верификации в БД и откройте ссылку:"
            echo "        psql \"\$DATABASE_URL\" -c \"SELECT verification_token FROM identity.users WHERE email='$email';\""
            ;;
        409)
            echo "    ! Email already registered"
            echo "    └─ Use: $0 login $email <password>"
            ;;
        *)
            err_msg=$(echo "$body" | jq -r '.error // empty')
            echo "    ✗ Registration failed (HTTP $code): ${err_msg:-$body}" >&2
            return 1
            ;;
    esac
}

# ==============================================================================
# Авторизация (логин)
# ==============================================================================
login() {
    local email="$1"
    local password="$2"

    echo "==> Logging in as: $email"
    response=$(curl -s -w "\n%{http_code}" -X POST "$IDENTITY_URL/login" \
        -H "Content-Type: application/json" \
        -d "$(cat <<EOF
{"email": "$email", "password": "$password"}
EOF
)")

    body=$(echo "$response" | sed '$d')
    code=$(echo "$response" | tail -1)

    case "$code" in
        200)
            token=$(echo "$body" | jq -r '.token')
            refresh_token=$(echo "$body" | jq -r '.refresh_token')
            echo "$token" > "$TOKEN_FILE"
            echo "$refresh_token" > "$REFRESH_TOKEN_FILE"
            echo "    ✓ Tokens saved to $TOKEN_FILE and $REFRESH_TOKEN_FILE"
            echo ""
            echo "    Используйте токен для запросов к lidar API:"
            echo "        curl -H 'Authorization: Bearer $token' http://localhost:8091/api/v1/tasks/<task-id>"
            echo ""
            echo "    Или экспортируйте в текущую сессию:"
            echo "        export TOKEN=$token"
            export TOKEN="$token"
            ;;
        401)
            err_msg=$(echo "$body" | jq -r '.error // empty')
            echo "    ✗ Login failed: ${err_msg:-unauthorized}" >&2
            return 1
            ;;
        403)
            echo "    ✗ Account not verified" >&2
            echo "    └─ Запрос на подтверждение отправлен на почту $email"
            echo "       Если SMTP не настроен, подтвердите вручную через psql."
            return 1
            ;;
        *)
            err_msg=$(echo "$body" | jq -r '.error // empty')
            echo "    ✗ Login failed (HTTP $code): ${err_msg:-$body}" >&2
            return 1
            ;;
    esac
}

# ==============================================================================
# Обновление пары токенов (refresh)
# ==============================================================================
refresh() {
    local refresh_token
    if [[ -f "$REFRESH_TOKEN_FILE" ]]; then
        refresh_token=$(cat "$REFRESH_TOKEN_FILE")
    else
        echo "No refresh token found. Run '$0 login <email> <password>' first." >&2
        return 1
    fi

    echo "==> Refreshing tokens"
    response=$(curl -s -w "\n%{http_code}" -X POST "$IDENTITY_URL/refresh" \
        -H "Content-Type: application/json" \
        -d "$(cat <<EOF
{"refresh_token": "$refresh_token"}
EOF
)")

    body=$(echo "$response" | sed '$d')
    code=$(echo "$response" | tail -1)

    case "$code" in
        200)
            token=$(echo "$body" | jq -r '.token')
            new_refresh=$(echo "$body" | jq -r '.refresh_token')
            echo "$token" > "$TOKEN_FILE"
            echo "$new_refresh" > "$REFRESH_TOKEN_FILE"
            echo "    ✓ Tokens refreshed"
            export TOKEN="$token"
            ;;
        401)
            err_msg=$(echo "$body" | jq -r '.error // empty')
            echo "    ✗ Refresh failed: ${err_msg:-invalid refresh token}" >&2
            return 1
            ;;
        *)
            err_msg=$(echo "$body" | jq -r '.error // empty')
            echo "    ✗ Refresh failed (HTTP $code): ${err_msg:-$body}" >&2
            return 1
            ;;
    esac
}

# ==============================================================================
# Показать сохранённый токен
# ==============================================================================
show_token() {
    if [[ -f "$TOKEN_FILE" ]]; then
        token=$(cat "$TOKEN_FILE")
        echo "$token"
    else
        echo "No token found. Run '$0 login <email> <password>' first." >&2
        return 1
    fi
}

show_refresh_token() {
    if [[ -f "$REFRESH_TOKEN_FILE" ]]; then
        cat "$REFRESH_TOKEN_FILE"
    else
        echo "No refresh token found. Run '$0 login <email> <password>' first." >&2
        return 1
    fi
}

# ==============================================================================
# Главная логика
# ==============================================================================
main() {
    cmd="${1:-help}"
    shift || true

    # Убедиться, что curl и jq установлены
    for cmd_req in curl jq; do
        if ! command -v "$cmd_req" &>/dev/null; then
            echo "Error: $cmd_req is required but not installed." >&2
            exit 1
        fi
    done

    case "$cmd" in
        register)
            [[ $# -lt 2 ]] && { echo "Usage: $0 register <email> <password>" >&2; exit 1; }
            register "$1" "$2"
            ;;
        login)
            [[ $# -lt 2 ]] && { echo "Usage: $0 login <email> <password>" >&2; exit 1; }
            login "$1" "$2"
            ;;
        full)
            [[ $# -lt 2 ]] && { echo "Usage: $0 full <email> <password>" >&2; exit 1; }
            register "$1" "$2" || true  # не падаем, если уже зарегистрирован
            echo ""
            login "$1" "$2"
            ;;
        token)
            show_token
            ;;
        refresh)
            refresh
            ;;
        refresh-token)
            show_refresh_token
            ;;
        help|--help|-h)
            awk '/^# ──/{if(p)exit;p=1} p' "$0" | sed 's/^# //; s/^#$//'
            ;;
        *)
            echo "Unknown command: $cmd" >&2
            echo "Usage: $0 [register|login|full|refresh|token|refresh-token|help]" >&2
            exit 1
            ;;
    esac
}

main "$@"
