import { Workspace, spawnChildJob, runStep } from "runtime/core.ts";
import { readSecrets } from "runtime/secrets.ts";
import * as Docker from "pkg/buildy/docker@1.0/mod.ts";

async function buildGoImage(ws: Workspace) {
  await Docker.buildImage({
    tag: `bristle/bristle:${ws.sha}`,
    dockerfilePath: `Dockerfile`,
  });
}

async function pushGoImage(ws: Workspace) {
  const [dockerHubUser, dockerHubToken] = await readSecrets(
    "DOCKER_HUB_USER",
    "DOCKER_HUB_TOKEN",
  );

  await await Docker.pushImage(`bristle/bristle:${ws.sha}`, "uplol", {
    tag: `bristle:${ws.sha}`,
    username: dockerHubUser,
    password: dockerHubToken,
  });
}

export async function buildBristle(ws: Workspace) {
  await runStep(buildGoImage, { name: `Build Bristle Image` });
  await runStep(pushGoImage, {
    name: `Push Bristle Image`,
    skipLocal: true,
  });
}

export async function run(ws: Workspace) {
  await spawnChildJob(`.ci/pipeline.ts:buildBristle`, {
    alias: `Build Bristle`,
  })
}