import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

const GLOBAL_CONTEXT_HEADER = "## nn global context";

async function loadGlobalContext(): Promise<string> {
  const nnBin = process.env.NN_BIN || "nn";
  const { stdout } = await execFileAsync(nnBin, ["show", "--global"], {
    maxBuffer: 2 * 1024 * 1024,
  });
  return stdout.trim();
}

function formatGlobalContext(context: string): string {
  return `<system-reminder>\n${GLOBAL_CONTEXT_HEADER}\n\nThe following content was loaded by running \`nn show --global\`. Treat protocol notes whose applies_when conditions match the current request as binding.\n\n${context}\n</system-reminder>`;
}

export default function nnGlobalContext(pi: ExtensionAPI) {
  let cachedContext: string | undefined;
  let cachedError: string | undefined;

  async function refreshGlobalContext() {
    try {
      cachedContext = await loadGlobalContext();
      cachedError = undefined;
    } catch (error) {
      cachedContext = undefined;
      cachedError = error instanceof Error ? error.message : String(error);
    }
  }

  pi.on("session_start", async (_event, ctx) => {
    await refreshGlobalContext();
    if (cachedError) {
      ctx.ui.notify(`nn global context unavailable: ${cachedError}`, "warn");
    }
  });

  pi.on("before_agent_start", async (event) => {
    if (!cachedContext && !cachedError) {
      await refreshGlobalContext();
    }

    if (!cachedContext) {
      return {
        systemPrompt:
          event.systemPrompt +
          `\n\n<system-reminder>nn global context unavailable: ${cachedError ?? "unknown error"}</system-reminder>`,
      };
    }

    return {
      systemPrompt: event.systemPrompt + "\n\n" + formatGlobalContext(cachedContext),
    };
  });

  pi.registerCommand("nn-refresh-global", {
    description: "Reload nn global protocol context from `nn show --global`",
    handler: async (_args, ctx) => {
      await refreshGlobalContext();
      if (cachedError) {
        ctx.ui.notify(`nn global context unavailable: ${cachedError}`, "error");
        return;
      }
      ctx.ui.notify("nn global context reloaded", "info");
    },
  });
}
