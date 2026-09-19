package scanner

// Consent handling.
//
// A consent layer covers the page, and axe rates what is visible — so without this a
// score describes the banner and not the site. An authority whose banner happens to be
// clean would look clean.
//
// What a careful visitor does is what we do: decline first. The site has to be usable
// without consent, declining sets no tracking, and it is the state most visitors who
// care end up in. Only where nothing can be declined is the accepting button used,
// because a visitor has to click something to see the page at all. Which path was
// taken is recorded, and a banner that survives both is recorded too — nobody should
// read a clean banner as a clean site.

// consentScript looks for the layer, tries to dismiss it and reports what happened.
// It searches through shadow roots, because the common consent tools render there.
const consentScript = `(() => {
  // Three tiers, tried in order. An explicit refusal first; then a button that merely
  // confirms the selection as it stands, which in these tools means nothing beyond the
  // necessary cookies; and only then consent, because a visitor has to click something
  // to reach the page at all.
  //
  // The wordings come from what German authorities actually ship: bund.de says "No, I
  // do not agree to statistical cookies", bundesregierung.de offers "Alle auswählen"
  // next to "Auswahl bestätigen" with nothing preselected. Patterns written for
  // "reject"/"ablehnen" alone walk past both.
  const refuse = [
    /nur\s+(technisch\s+)?(notwendige|erforderliche|essenzielle|essentielle)/i,
    /only\s+(necessary|essential)/i, /essential\s+only/i,
    /alle\s+ablehnen/i, /\bablehnen\b/i, /\bablehnung\b/i,
    /nicht\s+(zustimmen|einverstanden|akzeptieren)/i, /\bwiderspre/i,
    /\bnein\b/i, /\bno[,.]?\s/i, /do\s+not\s+agree/i, /don'?t\s+agree/i,
    /reject\s+all/i, /\breject\b/i, /\bdecline\b/i, /\bdeny\b/i,
    /weiter\s+ohne/i, /continue\s+without/i, /ohne\s+einwilligung/i,
  ];
  const confirmSelection = [
    /auswahl\s+(bestätigen|speichern|übernehmen)/i,
    /einstellungen\s+(speichern|bestätigen)/i,
    /save\s+(and\s+)?(close|settings|selection)/i,
    /confirm\s+(my\s+)?(choice|selection)/i,
    /^\s*(speichern|bestätigen|übernehmen)\s*$/i,
  ];
  const accept = [
    /alle\s+akzeptieren/i, /akzeptieren/i, /alle\s+(zulassen|auswählen)/i, /zustimmen/i,
    /einverstanden/i, /verstanden/i, /\bja\b/i, /\byes\b/i,
    /accept\s+all/i, /\baccept\b/i, /\bagree\b/i, /got\s+it/i, /allow\s+all/i,
  ];
  const consentWords = /cookie|einwillig|zustimm|consent|datenschutzeinstellung|privacy\s+settings|tracking/i;

  // Known consent tools. Their containers are recognised even when the wording is
  // unusual, which German authorities' own implementations often are.
  const knownContainers = [
    '#CybotCookiebotDialog', '#usercentrics-root', '#uc-center-container',
    '#BorlabsCookieBox', '.borlabs-cookie', '#onetrust-banner-sdk', '#onetrust-consent-sdk',
    '#cookiescript_injected', '#cmpbox', '.cmp-container', '#klaro', '.klaro',
    '#cookie-law-info-bar', '#cookieman-modal', '.cookie-consent', '#cookie-consent',
    '#cookie-banner', '.cookie-banner', '#cookiebanner', '#cookiehinweis',
    '[id*="cookie" i][class*="banner" i]', '[aria-label*="Cookie" i]',
  ];

  const roots = () => {
    const found = [document];
    const walk = (root) => {
      for (const el of root.querySelectorAll('*')) {
        if (el.shadowRoot) {
          found.push(el.shadowRoot);
          walk(el.shadowRoot);
        }
      }
    };
    walk(document);
    return found;
  };

  const all = (selector) => {
    const out = [];
    for (const root of roots()) {
      try {
        out.push(...root.querySelectorAll(selector));
      } catch { /* ein ungültiger Selektor darf den Rest nicht verhindern */ }
    }
    return out;
  };

  const visible = (el) => {
    const rect = el.getBoundingClientRect();
    if (rect.width < 2 || rect.height < 2) return false;
    const style = getComputedStyle(el);
    return style.visibility !== 'hidden' && style.display !== 'none' && style.opacity !== '0';
  };

  // A layer is what floats above the page, covers a good part of it and talks about
  // cookies or consent.
  const overlays = () => {
    const out = [];
    for (const root of roots()) {
      for (const el of root.querySelectorAll('div, section, aside, dialog, [role="dialog"], [role="alertdialog"]')) {
        const style = getComputedStyle(el);
        const fixed = style.position === 'fixed' || style.position === 'sticky';
        if (!fixed && el.tagName !== 'DIALOG' && !el.matches('[role="dialog"], [role="alertdialog"]')) continue;
        if (!visible(el)) continue;
        const rect = el.getBoundingClientRect();
        const area = (rect.width * rect.height) / (innerWidth * innerHeight);
        if (area < 0.04) continue;
        if (!consentWords.test(el.textContent || '')) continue;
        out.push(el);
      }
    }
    for (const selector of knownContainers) {
      for (const el of all(selector)) {
        if (visible(el) && !out.includes(el)) out.push(el);
      }
    }
    return out;
  };

  const present = overlays();
  if (present.length === 0) {
    return JSON.stringify({ state: 'none' });
  }

  const label = (el) =>
    [el.innerText, el.textContent, el.getAttribute('aria-label'), el.value, el.title]
      .filter(Boolean).join(' ').trim();

  const buttons = () => {
    const out = [];
    for (const el of all('button, a[href], input[type="button"], input[type="submit"], [role="button"]')) {
      if (visible(el)) out.push(el);
    }
    return out;
  };

  const inLayer = (el) => present.some((layer) => layer.contains(el) || layer === el);

  const pick = (patterns) => {
    const candidates = buttons().filter((el) => inLayer(el) || consentWords.test(label(el)));
    for (const pattern of patterns) {
      const hit = candidates.find((el) => pattern.test(label(el)));
      if (hit) return hit;
    }
    return null;
  };

  // A refusal expressed through markup rather than wording — some tools label the
  // button with an icon and carry the meaning in the class.
  const byAttribute = () =>
    buttons().find(
      (el) =>
        inLayer(el) &&
        /reject|deny|decline|refuse|necessary-only|only-necessary|ablehnen/i.test(
          [el.id, (el.className || '').toString(), el.getAttribute('data-testid') || ''].join(' '),
        ),
    ) ?? null;

  const refused = pick(refuse) ?? byAttribute() ?? pick(confirmSelection);
  const target = refused ?? pick(accept);
  if (!target) {
    return JSON.stringify({ state: 'blocked', reason: 'no button found' });
  }

  target.click();
  return JSON.stringify({ state: refused ? 'declined' : 'accepted', label: label(target).slice(0, 80) });
})()`

// consentCheckScript reports whether a layer is still in the way after the click.
const consentCheckScript = `(() => {
  const consentWords = /cookie|einwillig|zustimm|consent|datenschutzeinstellung|privacy\s+settings|tracking/i;
  const roots = () => {
    const found = [document];
    const walk = (root) => {
      for (const el of root.querySelectorAll('*')) {
        if (el.shadowRoot) { found.push(el.shadowRoot); walk(el.shadowRoot); }
      }
    };
    walk(document);
    return found;
  };
  for (const root of roots()) {
    for (const el of root.querySelectorAll('div, section, aside, dialog, [role="dialog"], [role="alertdialog"]')) {
      const style = getComputedStyle(el);
      const floating = style.position === 'fixed' || style.position === 'sticky' ||
        el.tagName === 'DIALOG' || el.matches('[role="dialog"], [role="alertdialog"]');
      if (!floating) continue;
      if (style.visibility === 'hidden' || style.display === 'none' || style.opacity === '0') continue;
      const rect = el.getBoundingClientRect();
      if (rect.width < 2 || rect.height < 2) continue;
      if ((rect.width * rect.height) / (innerWidth * innerHeight) < 0.04) continue;
      if (!consentWords.test(el.textContent || '')) continue;
      return JSON.stringify({ blocked: true });
    }
  }
  return JSON.stringify({ blocked: false });
})()`
