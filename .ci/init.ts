import { Workspace, registerPlugin } from "runtime/core.ts";
import { GithubCheckRunPlugin } from "pkg/buildy/github@1/plugins.ts";

export async function setup(ws: Workspace) {
  registerPlugin(
    new GithubCheckRunPlugin({
      repositorySlug: "uplol/bristle",
    })
  );
}