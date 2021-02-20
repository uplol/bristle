import { Workspace, spawnChildJob, runStep } from "runtime/core.ts";
import { readSecrets } from "runtime/secrets.ts";
import * as Docker from "pkg/buildy/docker@1.0/mod.ts";

async function buildGoImage(ws: Workspace, project: string) {
  await Docker.buildImage({
    tag: `bristle/${project}:${ws.sha}`,
    dockerfilePath: `${project}/Dockerfile`,
  });
}

async function pushGoImage(ws: Workspace, project: string) {
  const [dockerHubUser, dockerHubToken] = await readSecrets(
    "DOCKER_HUB_USER",
    "DOCKER_HUB_TOKEN",
  );

  await await Docker.pushImage(`bristle/${project}:${ws.sha}`, "uplol", {
    tag: `bristle-${project}:${ws.sha}`,
    username: dockerHubUser,
    password: dockerHubToken,
  });
}

export async function buildGo(ws: Workspace) {
  const project = ws.arg<string>("project");

  await runStep(buildGoImage, { name: `Build ${project} Image`, args: [project]});
  await runStep(pushGoImage, {
    name: `Push ${project} Image`,
    args: [project],
    skipLocal: true,
  });
}

export async function run(ws: Workspace) {
  for (const goProject of ["ingest", "forward"]) {
    await spawnChildJob(`.ci/pipeline.ts:buildGo`, {
      alias: `Build ${goProject}`,
      args: {
        project: goProject,
      }
    })
  }
}