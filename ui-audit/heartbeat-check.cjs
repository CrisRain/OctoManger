const { chromium } = require("/usr/local/lib/node_modules/playwright");

async function run() {
  const base = "http://127.0.0.1:5173";
  const res = await fetch(`${base}/api/v2/agents`);
  const json = await res.json();
  const firstAgent = json?.items?.[0];
  if (!firstAgent) {
    throw new Error("no agent found from /api/v2/agents");
  }

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  await page.goto(`${base}/logs`, { waitUntil: "networkidle" });
  await page.locator(".arco-tabs-tab").filter({ hasText: "Agent 事件射频流" }).first().click();

  const firstAgentCard = page.locator(".split-list .list-item").first();
  await firstAgentCard.waitFor({ state: "visible", timeout: 10000 });
  await firstAgentCard.click();
  await page.waitForSelector(".status-led.live", { timeout: 10000 });
  await page.waitForTimeout(6500);

  const logsAnalysis = await page.evaluate(() => {
    const messages = Array.from(document.querySelectorAll(".log-row .log-message"))
      .map((el) => (el.textContent || "").trim())
      .filter(Boolean);

    const leakedHeartbeatRows = messages.filter((m) =>
      m.includes("\"runtime_state\"") ||
      m.includes("\"desired_state\"") ||
      m.includes("\"last_heartbeat_at\"")
    );

    return {
      liveLedCount: document.querySelectorAll(".status-led.live").length,
      leakedHeartbeatRows: leakedHeartbeatRows.length,
      logRows: messages.length,
      sampleLeak: leakedHeartbeatRows.slice(0, 3),
    };
  });

  await page.goto(`${base}/agents/${firstAgent.id}`, { waitUntil: "networkidle" });
  await page.waitForSelector(".stream-badge", { timeout: 10000 });
  await page.waitForTimeout(6500);

  const detailAnalysis = await page.evaluate(() => {
    const streamBadge = document.querySelector(".stream-badge");
    const runtimeTag = document.querySelector(".status-tag-pill");
    return {
      streamText: (streamBadge?.textContent || "").trim(),
      streamClass: streamBadge?.className || "",
      runtimeText: (runtimeTag?.textContent || "").trim(),
    };
  });

  await browser.close();

  const output = {
    agentId: firstAgent.id,
    logsAnalysis,
    detailAnalysis,
  };
  console.log(JSON.stringify(output, null, 2));
}

run().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
