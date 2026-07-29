import { initRouter, route } from "./router.js";
import { renderLogin } from "./pages/login.js";
import { renderRegister } from "./pages/register.js";
import { renderVerified } from "./pages/verified.js";
import { renderExperiments } from "./pages/experiments.js";
import { renderExperimentDetail } from "./pages/experiment-detail.js";
import { renderUpload } from "./pages/upload.js";

// Register routes
route("/login", renderLogin);
route("/register", renderRegister);
route("/verified", renderVerified);
route("/experiments", renderExperiments);
route("/experiments/:id", renderExperimentDetail);
route("/upload", renderUpload);

// Start the router
initRouter();
