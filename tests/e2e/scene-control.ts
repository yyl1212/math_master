type SceneRequest = {
  post: (url: string, options: { headers: Record<string, string>; timeout: number }) => Promise<{ status: () => number }>;
};
type SceneRuntime = { controlURL: string; token: string };

export async function changeScene(request: SceneRequest, runtime: SceneRuntime, name: string) {
  try {
    const response = await request.post(runtime.controlURL + "/scene/" + name, {
      headers: { Authorization: "Bearer " + runtime.token },
      // The complete fixture publishes 30 lessons and five banks through the
      // real workflows. This setup-only limit stays below the 30s test budget;
      // ordinary scenes keep their original limit and no writes are retried.
      timeout: name === "content-acceptance" ? 10000 : 5000,
    });
    if (response.status() !== 204) throw new Error("Test scene change failed");
  } catch (error) {
    // Do not retain a transport cause: it can contain the private control URL
    // and authorization token. Report only a safe timeout classification.
    if (error instanceof Error && (error.name === "TimeoutError" || /Timeout \d+ms exceeded/.test(error.message))) {
      throw new Error("Test scene change timed out");
    }
    throw new Error("Test scene change failed");
  }
}
