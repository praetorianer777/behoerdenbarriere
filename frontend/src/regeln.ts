/**
 * Deutsche Namen für die Regeln, die axe-core prüft, und je ein Satz dazu, wen der
 * Befund ausschließt.
 *
 * Die Namen kommen sonst aus der Antwort der API und sind englische Fachsätze
 * ("Elements must only use supported ARIA attributes"). Auf einer Seite über Zugang
 * ist ausgerechnet der Teil, der sagt, was zu tun ist, dann nicht lesbar.
 *
 * Die Wirkung ist wichtiger als die Regel: Ein Name benennt den Fehler, der Satz
 * darunter die Person, die davor steht. Beides bleibt bei der Sache — was axe findet,
 * ist ein Verdacht auf eine Barriere, keine erlebte.
 *
 * Die Tabelle deckt, was in den Prüfungen vorkommt, und die Regeln, die erfahrungsgemäß
 * als Nächstes auftauchen. Was fehlt, fällt auf den englischen Text der API zurück —
 * eine Lücke darf keine leere Zeile ergeben.
 */
export interface Regel {
  name: string
  wen: string
}

export const regeln: Record<string, Regel> = {
  'color-contrast': {
    name: 'Zu wenig Kontrast zwischen Text und Hintergrund',
    wen: 'Wer schlecht sieht oder bei Sonnenlicht auf das Telefon schaut, kann den Text nicht entziffern.',
  },
  'link-name': {
    name: 'Link ohne erkennbare Beschriftung',
    wen: 'Ein Screenreader liest nur „Link“ vor. Wohin er führt, bleibt offen.',
  },
  'aria-allowed-attr': {
    name: 'Unzulässige Zusatzangabe am Element',
    wen: 'Die Angaben für Screenreader passen nicht zum Element — vorgelesen wird etwas anderes, als dasteht.',
  },
  'aria-hidden-focus': {
    name: 'Verstecktes Element ist mit der Tastatur erreichbar',
    wen: 'Wer mit der Tastatur navigiert, landet auf etwas, das der Screenreader nicht vorliest.',
  },
  'aria-required-children': {
    name: 'Angekündigte Struktur ohne die zugehörigen Einträge',
    wen: 'Ein Menü oder eine Liste wird angesagt, hat aber nichts, was sich vorlesen ließe.',
  },
  'aria-required-parent': {
    name: 'Eintrag außerhalb der Struktur, zu der er gehört',
    wen: 'Ein Menüeintrag ohne Menü drumherum wird falsch oder gar nicht angesagt.',
  },
  'aria-required-attr': {
    name: 'Fehlende Pflichtangabe am Bedienelement',
    wen: 'Ob etwas aus- oder eingeklappt, an- oder ausgeschaltet ist, wird nicht mitgeteilt.',
  },
  'aria-prohibited-attr': {
    name: 'An diesem Element nicht erlaubte Zusatzangabe',
    wen: 'Die Angabe wird entweder ignoriert oder sie überschreibt den sichtbaren Text.',
  },
  'aria-valid-attr': {
    name: 'Unbekannte Zusatzangabe',
    wen: 'Meist ein Tippfehler im Attributnamen. Die Angabe wirkt dann gar nicht.',
  },
  'aria-valid-attr-value': {
    name: 'Ungültiger Wert in einer Zusatzangabe',
    wen: 'Der Verweis geht ins Leere; die Beschriftung, auf die er zeigt, wird nicht vorgelesen.',
  },
  'aria-conditional-attr': {
    name: 'Zusatzangabe passt nicht zum Zustand des Elements',
    wen: 'Angesagt wird ein Zustand, in dem das Element gar nicht ist.',
  },
  'aria-roles': {
    name: 'Unbekannte Rolle am Element',
    wen: 'Der Screenreader weiß nicht, als was er das Element ankündigen soll.',
  },
  'aria-command-name': {
    name: 'Bedienelement ohne Beschriftung',
    wen: 'Angesagt wird nur, dass man etwas auslösen kann — nicht, was.',
  },
  'aria-input-field-name': {
    name: 'Eingabefeld ohne Beschriftung',
    wen: 'Wer nicht sieht, erfährt nicht, was in das Feld gehört.',
  },
  'aria-toggle-field-name': {
    name: 'Schalter ohne Beschriftung',
    wen: 'Dass sich etwas an- und ausschalten lässt, wird angesagt — wofür, nicht.',
  },
  'aria-dialog-name': {
    name: 'Dialogfenster ohne Titel',
    wen: 'Ein Fenster öffnet sich, und der Screenreader kann nicht sagen, worum es darin geht.',
  },
  'aria-tooltip-name': { name: 'Kurzhinweis ohne Text', wen: 'Der Hinweis bleibt stumm.' },
  'aria-meter-name': {
    name: 'Anzeige ohne Beschriftung',
    wen: 'Ein Wert wird angesagt, ohne zu sagen, wovon.',
  },
  'aria-progressbar-name': {
    name: 'Fortschrittsanzeige ohne Beschriftung',
    wen: 'Angesagt wird ein Fortschritt, ohne wobei.',
  },
  'aria-treeitem-name': {
    name: 'Eintrag im Baum ohne Beschriftung',
    wen: 'Der Eintrag lässt sich ansteuern, aber nicht vorlesen.',
  },
  'aria-text': {
    name: 'Als Text ausgezeichneter Bereich enthält Bedienelemente',
    wen: 'Was darin bedienbar ist, wird für Screenreader zu bloßem Text.',
  },
  'aria-braille-equivalent': {
    name: 'Braille-Beschriftung ohne dazugehörige Textbeschriftung',
    wen: 'Auf einer Braillezeile steht etwas anderes als im Ton des Screenreaders.',
  },
  'link-in-text-block': {
    name: 'Link im Fließtext nur an der Farbe zu erkennen',
    wen: 'Wer Farben schlecht unterscheidet, sieht nicht, dass dort überhaupt ein Link steht.',
  },
  'image-alt': {
    name: 'Bild ohne Alternativtext',
    wen: 'Wer nicht sieht, erfährt nicht, was auf dem Bild zu sehen ist.',
  },
  'svg-img-alt': {
    name: 'Grafik ohne Alternativtext',
    wen: 'Betrifft oft Symbole, die allein die Bedeutung tragen — etwa ein Briefumschlag für „Kontakt“.',
  },
  'role-img-alt': {
    name: 'Als Bild ausgezeichnetes Element ohne Alternativtext',
    wen: 'Angekündigt wird ein Bild, dessen Inhalt niemand erfährt.',
  },
  'input-image-alt': {
    name: 'Schaltfläche aus einem Bild ohne Alternativtext',
    wen: 'Oft der Absende-Knopf eines Formulars — er lässt sich finden, aber nicht deuten.',
  },
  'object-alt': {
    name: 'Eingebettetes Objekt ohne Alternativtext',
    wen: 'Was darin steckt, bleibt ohne Blick darauf unbekannt.',
  },
  'area-alt': {
    name: 'Anklickbare Bildfläche ohne Alternativtext',
    wen: 'Eine Landkarte mit klickbaren Regionen wird zu einer Fläche ohne Ziele.',
  },
  'button-name': {
    name: 'Schaltfläche ohne Beschriftung',
    wen: 'Vorgelesen wird „Schaltfläche“. Was sie auslöst, bleibt unklar.',
  },
  'input-button-name': {
    name: 'Schaltfläche ohne Beschriftung',
    wen: 'Vorgelesen wird „Schaltfläche“. Was sie auslöst, bleibt unklar.',
  },
  'select-name': {
    name: 'Auswahlfeld ohne Beschriftung',
    wen: 'Eine Liste zum Aufklappen, von der niemand erfährt, wonach sie fragt.',
  },
  label: {
    name: 'Eingabefeld ohne Beschriftung',
    wen: 'Wer nicht sieht, erfährt nicht, was in das Feld gehört — bei einem Antrag heißt das: gar nicht ausfüllen.',
  },
  'label-title-only': {
    name: 'Feld nur über den Mauszeiger-Hinweis beschriftet',
    wen: 'Wer mit der Tastatur oder dem Finger bedient, bekommt den Hinweis nie zu sehen.',
  },
  'form-field-multiple-labels': {
    name: 'Eingabefeld mit mehreren Beschriftungen',
    wen: 'Welche davon vorgelesen wird, hängt vom Programm ab.',
  },
  'autocomplete-valid': {
    name: 'Unbrauchbare Angabe zum automatischen Ausfüllen',
    wen: 'Das Formular füllt sich nicht von selbst. Wer mühsam tippt, tippt alles.',
  },
  'meta-viewport': {
    name: 'Zoomen ist gesperrt',
    wen: 'Wer die Seite größer ziehen muss, um sie zu lesen, kann es nicht.',
  },
  'meta-refresh': {
    name: 'Die Seite lädt sich von selbst neu',
    wen: 'Wer langsamer liest oder tippt, wird mitten im Vorgang unterbrochen.',
  },
  'frame-title': {
    name: 'Eingebetteter Rahmen ohne Titel',
    wen: 'Vorgelesen wird „Frame“ — meist steckt ein Video, eine Karte oder ein Formular darin.',
  },
  'frame-focusable-content': {
    name: 'Rahmen mit bedienbarem Inhalt ist für die Tastatur gesperrt',
    wen: 'Der Inhalt ist da, aber ohne Maus nicht erreichbar.',
  },
  'scrollable-region-focusable': {
    name: 'Scrollbarer Bereich ist mit der Tastatur nicht erreichbar',
    wen: 'Wer keine Maus benutzt, kommt an den Inhalt darin nicht heran.',
  },
  'nested-interactive': {
    name: 'Bedienelement in einem Bedienelement',
    wen: 'Screenreader und Tastatur erreichen nur eines von beiden.',
  },
  list: {
    name: 'Liste enthält Elemente, die nicht hineingehören',
    wen: 'Der Screenreader sagt eine Liste an, kann ihre Einträge aber nicht mehr zählen.',
  },
  listitem: {
    name: 'Listeneintrag ohne umgebende Liste',
    wen: 'Angesagt wird ein Eintrag, ohne wovon — die Gliederung geht verloren.',
  },
  'definition-list': {
    name: 'Begriffsliste falsch aufgebaut',
    wen: 'Welcher Erklärungstext zu welchem Begriff gehört, ist nicht mehr zu erkennen.',
  },
  dlitem: {
    name: 'Eintrag einer Begriffsliste ohne umgebende Liste',
    wen: 'Begriff und Erklärung stehen ohne Zusammenhang nebeneinander.',
  },
  'html-has-lang': {
    name: 'Seite ohne Sprachangabe',
    wen: 'Der Screenreader weiß nicht, welche Sprache er sprechen soll, und liest Deutsch mit englischer Aussprache.',
  },
  'html-lang-valid': {
    name: 'Ungültige Sprachangabe der Seite',
    wen: 'Die Angabe ist da, aber unbrauchbar — vorgelesen wird trotzdem in der falschen Sprache.',
  },
  'valid-lang': {
    name: 'Ungültige Sprachangabe an einem Abschnitt',
    wen: 'Ein fremdsprachiger Abschnitt wird nicht umgeschaltet.',
  },
  'document-title': {
    name: 'Seite ohne Titel',
    wen: 'Unter mehreren offenen Seiten ist diese nicht wiederzufinden.',
  },
  'heading-order': {
    name: 'Überschriftenebenen übersprungen',
    wen: 'Wer sich von Überschrift zu Überschrift bewegt, verliert die Gliederung der Seite.',
  },
  'empty-heading': {
    name: 'Leere Überschrift',
    wen: 'Die Gliederung kündigt einen Abschnitt an, der keinen Namen hat.',
  },
  'page-has-heading-one': {
    name: 'Seite ohne Hauptüberschrift',
    wen: 'Es fehlt der Punkt, zu dem man springt, um zu erfahren, worum es hier geht.',
  },
  bypass: {
    name: 'Kein Sprung zum Hauptinhalt',
    wen: 'Wer mit der Tastatur bedient, muss auf jeder Seite erst das ganze Menü durchgehen.',
  },
  region: {
    name: 'Inhalt liegt außerhalb der Seitenbereiche',
    wen: 'Wer direkt zum Hauptinhalt springen will, findet ihn nicht.',
  },
  'landmark-one-main': {
    name: 'Seite ohne gekennzeichneten Hauptinhalt',
    wen: 'Wer das Menü überspringen will, hat kein Ziel dafür.',
  },
  tabindex: {
    name: 'Tastaturreihenfolge künstlich verbogen',
    wen: 'Der Fokus springt an unerwartete Stellen, statt dem Text zu folgen.',
  },
  'td-headers-attr': {
    name: 'Tabellenzelle verweist auf eine Kopfzelle, die es nicht gibt',
    wen: 'Vorgelesen wird der Wert ohne die Spalte, zu der er gehört.',
  },
  'th-has-data-cells': {
    name: 'Tabellenkopf ohne zugeordnete Zellen',
    wen: 'Die Überschrift der Spalte wird beim Vorlesen der Werte nicht mitgenannt.',
  },
  'video-caption': {
    name: 'Video ohne Untertitel',
    wen: 'Wer nicht hört, bekommt vom Gesprochenen nichts mit.',
  },
  'no-autoplay-audio': {
    name: 'Ton startet von selbst',
    wen: 'Wer einen Screenreader hört, versteht ihn neben dem Ton nicht mehr.',
  },
  marquee: {
    name: 'Laufschrift',
    wen: 'Text, der wegläuft, ist beim langsamen Lesen nicht zu fassen.',
  },
  blink: { name: 'Blinkender Text', wen: 'Blinken kann Anfälle auslösen und lenkt jede ab.' },
  'server-side-image-map': {
    name: 'Bildkarte, die nur mit der Maus funktioniert',
    wen: 'Ohne Maus gibt es keinen Weg zu den Zielen darin.',
  },
  'avoid-inline-spacing': {
    name: 'Text-Abstände lassen sich nicht vergrößern',
    wen: 'Wer Zeilen- und Buchstabenabstand zum Lesen erhöht, bekommt eine zerbrochene Seite.',
  },
  'css-orientation-lock': {
    name: 'Seite erzwingt eine Bildschirmausrichtung',
    wen: 'Wer das Telefon an einem Rollstuhl fest montiert hat, kann es nicht drehen.',
  },
  'target-size': {
    name: 'Bedienfläche zu klein',
    wen: 'Wer die Hände nicht ruhig hält, trifft sie nicht.',
  },
  'duplicate-id-aria': {
    name: 'Kennung mehrfach vergeben',
    wen: 'Verweise auf Beschriftungen landen beim falschen Element.',
  },
  'summary-name': {
    name: 'Aufklappbarer Abschnitt ohne Beschriftung',
    wen: 'Dass sich etwas aufklappen lässt, wird angesagt — was darin steht, nicht.',
  },
}

/**
 * Der Name der Regel auf Deutsch, sonst der englische Text der API und zuletzt die
 * Kennung. Eine fehlende Übersetzung darf keine leere Zeile ergeben.
 */
export function regelName(rule: { rule_id: string; help?: string | null }): string {
  return regeln[rule.rule_id]?.name ?? rule.help ?? rule.rule_id
}

/** Wen der Befund ausschließt — leer, solange die Regel nicht übersetzt ist. */
export function regelWirkung(ruleId: string): string | undefined {
  return regeln[ruleId]?.wen
}
