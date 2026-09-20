import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import lighthouse from 'lighthouse'

const port = Number(process.env.PORT ?? 3000)
const chromePath = process.env.CHROME_PATH ?? '/usr/bin/chromium'
const chromePort = Number(process.env.CHROME_DEBUG_PORT ?? 9333)

// Lighthouse braucht einen Browser, den es selbst steuert: Es setzt Emulation,
// Netzwerkbedingungen und eigene Flags und teilt sich den Browser deshalb nicht mit
// dem Scanner. Also läuft hier ein eigener, im selben Container.
let chrome
function startChrome() {
  chrome = spawn(chromePath, [
    '--headless=new',
    '--no-sandbox',
    '--disable-gpu',
    '--disable-dev-shm-usage',
    `--remote-debugging-port=${chromePort}`,
    '--remote-debugging-address=127.0.0.1',
  ])
  chrome.on('exit', (code) => {
    console.error(`chrome beendet (${code}), wird neu gestartet`)
    setTimeout(startChrome, 1000)
  })
}

async function waitForChrome() {
  for (let attempt = 0; attempt < 60; attempt++) {
    try {
      const response = await fetch(`http://127.0.0.1:${chromePort}/json/version`)
      if (response.ok) return
    } catch {
      // noch nicht da
    }
    await new Promise((resolve) => setTimeout(resolve, 500))
  }
  throw new Error('chrome ist nicht hochgekommen')
}

async function audit(url) {
  const result = await lighthouse(url, {
    port: chromePort,
    output: 'json',
    logLevel: 'error',
    onlyCategories: ['accessibility'],
    // Ein Behördenauftritt wird überwiegend mobil aufgerufen; die Voreinstellung von
    // Lighthouse ist ohnehin die mobile Emulation, hier steht sie ausdrücklich.
    formFactor: 'mobile',
    screenEmulation: { mobile: true, width: 412, height: 823, deviceScaleFactor: 1.75 },
  })

  const lhr = result.lhr
  const category = lhr.categories.accessibility
  const failed = Object.values(lhr.audits)
    .filter((entry) => entry.score !== null && entry.score < 1 && entry.scoreDisplayMode !== 'notApplicable')
    .map((entry) => entry.id)
    .sort()

  return {
    // Lighthouse liefert 0..1; die Oberfläche rechnet in Punkten.
    score: category.score === null ? null : Math.round(category.score * 1000) / 10,
    failed_audits: failed,
    lighthouse_version: lhr.lighthouseVersion,
    fetched_url: lhr.finalDisplayedUrl ?? url,
  }
}

const server = createServer(async (req, res) => {
  if (req.method === 'GET' && req.url === '/healthz') {
    res.writeHead(200, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ status: 'ok' }))
    return
  }
  if (req.method !== 'POST' || !req.url.startsWith('/audit')) {
    res.writeHead(404).end()
    return
  }

  let body = ''
  for await (const chunk of req) {
    body += chunk
    if (body.length > 4096) {
      res.writeHead(413).end()
      return
    }
  }

  let url
  try {
    url = new URL(JSON.parse(body).url).toString()
  } catch (error) {
    res.writeHead(400, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ error: `unbrauchbare URL: ${error.message}` }))
    return
  }

  try {
    const result = await audit(url)
    res.writeHead(200, { 'content-type': 'application/json' })
    res.end(JSON.stringify(result))
  } catch (error) {
    // Ein gescheiterter Lauf ist kein Grund, den Scan zu verlieren: Der Aufrufer
    // bekommt den Fehler und speichert die eigene Zahl trotzdem.
    console.error(`audit ${url}: ${error.stack ?? error}`)
    res.writeHead(502, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ error: String(error.message ?? error) }))
  }
})

startChrome()
await waitForChrome()
server.listen(port, () => console.log(`lighthouse-dienst auf ${port}`))
