import type { Rule } from '../api/types'
import { impactLabel, principleLabel } from '../lib'

interface Props {
  rules: Rule[]
  heading: string
  emptyText: string
}

export function RuleList({ rules, heading, emptyText }: Props) {
  if (rules.length === 0) {
    return (
      <section>
        <h3 className="text-lg font-semibold">{heading}</h3>
        <p className="mt-2 text-slate-700">{emptyText}</p>
      </section>
    )
  }

  return (
    <section>
      <h3 className="text-lg font-semibold">{heading}</h3>
      <ul className="mt-2 space-y-3">
        {rules.map((rule) => (
          <li key={rule.rule_id} className="rounded-lg border border-slate-200 bg-white p-4">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <h4 className="font-semibold">{rule.help ?? rule.rule_id}</h4>
              <span className="text-sm text-slate-700">
                {impactLabel[rule.impact]} · {principleLabel[rule.principle] ?? rule.principle}
              </span>
            </div>
            <p className="mt-1 text-sm text-slate-700">
              Auf {rule.pages} {rule.pages === 1 ? 'Seite' : 'Seiten'}, {rule.nodes}{' '}
              {rule.nodes === 1 ? 'Element' : 'Elemente'} betroffen
            </p>
            {rule.sample_html && (
              <details className="mt-2">
                <summary className="cursor-pointer text-sm underline">Beispiel im Quelltext</summary>
                <pre className="mt-2 overflow-x-auto rounded bg-slate-100 p-3 text-xs">
                  <code>{rule.sample_html}</code>
                </pre>
              </details>
            )}
            {rule.help_url && (
              <p className="mt-2 text-sm">
                <a href={rule.help_url} className="underline">
                  Regel {rule.rule_id} nachlesen
                </a>
              </p>
            )}
          </li>
        ))}
      </ul>
    </section>
  )
}
