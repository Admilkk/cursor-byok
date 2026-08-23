import ReactDOM from "react-dom/client";
import "../node_modules/monaco-editor/min/vs/editor/editor.main.css";
import { App } from "./App";
import { setRuntimeLocale } from "./i18n/runtime";
import { appStore } from "./store/appStore";
import { applyTheme } from "./theme/theme";
import "./styles/globals.scss";

setRuntimeLocale("zh-CN");
applyTheme(appStore.getSnapshot().theme);
void appStore.refresh();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <App />,
);
