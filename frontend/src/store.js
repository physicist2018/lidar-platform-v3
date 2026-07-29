// Simple application state store.
// Triggers a re-render of the active page on change (pub/sub).

const listeners = new Set();

let state = {
  token: localStorage.getItem("token"),
  user: null,
  experiments: [],
  currentExperiment: null,
  tasks: [],
};

export function getState() {
  return state;
}

export function setState(partial) {
  state = { ...state, ...partial };
  listeners.forEach((fn) => fn(state));
}

export function subscribe(fn) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
