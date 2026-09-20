// Package maildns reads what a domain publishes about its email: the MX records that
// say which host receives it, the SPF record that says who may send in its name, and
// the DMARC record that says what to do with forgeries.
//
// Everything here comes from public DNS. Nothing is probed, no port is opened, no mail
// server is spoken to. We read what the domain publishes about itself.
//
// The classification is a reading of host names and nothing more. An MX record says
// which host accepts the mail — not who ends up reading it: a German processor may
// itself run on Microsoft, and an MX may point at a spam filter while the mailboxes sit
// somewhere else entirely. That distinction has to survive all the way to the page,
// which is why every record is kept in full beside its classification.
package maildns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/praetorianer777/behoerdenbarriere/internal/thirdparty"
)

// Provider is the operator a host name points to.
type Provider string

const (
	Microsoft365    Provider = "microsoft365"
	GoogleWorkspace Provider = "google"
	Proofpoint      Provider = "proofpoint"
	Barracuda       Provider = "barracuda"
	Cloudflare      Provider = "cloudflare"
	Mimecast        Provider = "mimecast"
	Hornetsecurity  Provider = "hornetsecurity"
	Sophos          Provider = "sophos"
	Symantec        Provider = "symantec"
	Telekom         Provider = "telekom"
	IONOS           Provider = "ionos"
	Strato          Provider = "strato"
	Hetzner         Provider = "hetzner"
	Netcup          Provider = "netcup"
	MailboxOrg      Provider = "mailbox-org"
	Retarus         Provider = "retarus"
	PublicIT        Provider = "public-it" // a public-sector IT provider (Dataport, ITZBund, …)
	SelfOperated    Provider = "self"      // the MX lies within the authority's own domain
	NoMail          Provider = "none"      // the domain publishes no MX at all
	UnknownProvider Provider = "unknown"
)

// USBased says whether a provider is under US jurisdiction. It is a statement about the
// company, not about where a particular server stands — and not, by itself, a legal
// finding.
var USBased = map[Provider]bool{
	Microsoft365: true, GoogleWorkspace: true, Proofpoint: true,
	Barracuda: true, Cloudflare: true, Mimecast: true, Symantec: true,
}

// Filters are services that sit in front of the mailboxes rather than holding them.
// The distinction matters: an MX pointing at a spam filter says where mail is screened,
// and says nothing at all about where it is kept. Reading such a record as "the mail
// runs there" would be exactly the overreach this data invites.
var Filters = map[Provider]bool{
	Proofpoint: true, Barracuda: true, Mimecast: true,
	Hornetsecurity: true, Sophos: true, Symantec: true,
}

// rules map a host suffix onto its operator, first match wins.
var rules = []struct {
	suffixes []string
	provider Provider
}{
	{[]string{"mail.protection.outlook.com", "outlook.com", "office365.com",
		"protection.outlook.com", "exchangelabs.com"}, Microsoft365},
	{[]string{"google.com", "googlemail.com", "aspmx.l.google.com", "psmtp.com"}, GoogleWorkspace},
	{[]string{"pphosted.com", "ppe-hosted.com", "proofpoint.com"}, Proofpoint},
	{[]string{"barracudanetworks.com", "ess.barracuda.com", "barracuda.com"}, Barracuda},
	{[]string{"mimecast.com", "mimecast.de"}, Mimecast},
	{[]string{"hornetsecurity.com", "antispameurope.com", "antispameurope.de"}, Hornetsecurity},
	{[]string{"sophos.com", "sophos.de", "reflexion.net"}, Sophos},
	{[]string{"messagelabs.com", "symanteccloud.com"}, Symantec},
	{[]string{"mx.cloudflare.net", "cloudflare.net", "cloudflare.com"}, Cloudflare},
	{[]string{"t-online.de", "telekom.de", "t-systems.com", "t-ipnet.de"}, Telekom},
	{[]string{"kundenserver.de", "ionos.de", "ionos.com", "1und1.de", "perfora.net",
		"schlund.de"}, IONOS},
	{[]string{"strato.de", "rzone.de", "cronon.net"}, Strato},
	{[]string{"your-server.de", "hetzner.de", "hetzner.com"}, Hetzner},
	{[]string{"netcup.net", "netcup.de"}, Netcup},
	{[]string{"mailbox.org", "heinlein-support.de", "heinlein-hosting.de"}, MailboxOrg},
	{[]string{"retarus.de", "retarus.com"}, Retarus},
	// Public-sector IT providers. The list is short on purpose: every wrong entry here
	// turns a German authority into a foreign cloud or the other way round.
	{[]string{"dataport.de", "itzbund.de", "komm.one", "krzn.de", "lvn.niedersachsen.de",
		"citeq.de", "kdo.de", "ekom21.de", "akdb.de", "zit-bb.de", "ozg-cloud.de",
		"bundesdruckerei.de", "govdata.de", "itk-rheinland.de", "lecos-gmbh.de",
		"regio-it.de", "regioit-aachen.de", "krz.de", "kvnbw.de", "landsh.de",
		"kgrz-ks.de", "civitec.de", "kdvz-frechen.de", "dfn.de", "drv-bund.de",
		"dbtg.de", "bfinv.de"}, PublicIT},
}

// Classify names the operator behind a host.
func Classify(host string) Provider {
	host = normalizeHost(host)
	if host == "" {
		return UnknownProvider
	}
	for _, rule := range rules {
		for _, suffix := range rule.suffixes {
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return rule.provider
			}
		}
	}
	return UnknownProvider
}

// MXHost is one mail exchanger with its preference.
type MXHost struct {
	Host       string   `json:"host"`
	Preference uint16   `json:"preference"`
	Provider   Provider `json:"provider"`
}

// Record is what one domain publishes.
type Record struct {
	Domain string `json:"domain"`
	// MX holds every mail exchanger, in the order the domain prefers them.
	MX []MXHost `json:"mx,omitempty"`
	// Provider is the operator of the most-preferred MX — the host that receives the
	// mail. Nothing more: what happens to it afterwards is invisible from here.
	Provider Provider `json:"provider"`
	// SPF is the record as published, kept in full so a reader can check our reading.
	SPF string `json:"spf,omitempty"`
	// SPFIncludes are the domains the record authorises to send. This is a permission
	// to send, not a place where mail is kept: a newsletter tool or a ticket system
	// turns up here without ever seeing an inbox.
	SPFIncludes []string   `json:"spf_includes,omitempty"`
	SPFSenders  []Provider `json:"spf_senders,omitempty"`
	DMARC       string     `json:"dmarc,omitempty"`
	// DMARCPolicy is p= from the DMARC record: none, quarantine or reject.
	DMARCPolicy string    `json:"dmarc_policy,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
	Err         string    `json:"error,omitempty"`
}

// Resolver is the part of net.Resolver this package uses, so a test does not need DNS.
type Resolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

type Client struct {
	resolver Resolver
	timeout  time.Duration
}

func New(resolver Resolver, timeout time.Duration) *Client {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{resolver: resolver, timeout: timeout}
}

// Lookup reads MX, SPF and DMARC for one domain. A domain without MX is not an error:
// plenty of authority web domains carry no mail at all, and saying so is a result.
func (c *Client) Lookup(ctx context.Context, domain string) Record {
	domain = normalizeHost(domain)
	record := Record{Domain: domain, Provider: UnknownProvider, CheckedAt: time.Now().UTC()}
	if domain == "" {
		record.Err = "no domain"
		return record
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	mx, err := c.resolver.LookupMX(ctx, domain)
	switch {
	case isNotFound(err):
		record.Provider = NoMail
	case err != nil:
		record.Err = err.Error()
	default:
		record.MX = toHosts(mx)
		if len(record.MX) == 0 {
			record.Provider = NoMail
		} else {
			record.Provider = provider(record)
		}
	}

	if txt, err := c.resolver.LookupTXT(ctx, domain); err == nil {
		record.SPF = findSPF(txt)
		record.SPFIncludes = spfIncludes(record.SPF)
		record.SPFSenders = classifyAll(record.SPFIncludes)
	}
	if txt, err := c.resolver.LookupTXT(ctx, "_dmarc."+domain); err == nil {
		record.DMARC = findDMARC(txt)
		record.DMARCPolicy = dmarcPolicy(record.DMARC)
	}
	return record
}

// provider reads the operator out of the mail exchangers: the most-preferred one
// decides, because that is the host the mail actually reaches.
//
// Where no known operator is behind it, two cases remain worth telling apart from
// "unknown": every exchanger inside the authority's own domain — the house runs its own
// mail — and an exchanger under a domain of the public sector, which is the shared IT
// of a state or the federal government. Both look like nothing to a suffix list and are
// exactly the cases this data gets quoted for.
func provider(record Record) Provider {
	if known := record.MX[0].Provider; known != UnknownProvider {
		return known
	}
	if selfOperated(record) {
		return SelfOperated
	}
	if thirdparty.PublicBody(record.MX[0].Host) {
		return PublicIT
	}
	return UnknownProvider
}

func selfOperated(record Record) bool {
	for _, mx := range record.MX {
		if mx.Host != record.Domain && !strings.HasSuffix(mx.Host, "."+record.Domain) {
			return false
		}
	}
	return true
}

func toHosts(mx []*net.MX) []MXHost {
	out := make([]MXHost, 0, len(mx))
	for _, entry := range mx {
		host := normalizeHost(entry.Host)
		// "." as the whole record is the null MX of RFC 7505: the domain says it takes
		// no mail at all.
		if host == "" {
			continue
		}
		out = append(out, MXHost{Host: host, Preference: entry.Pref, Provider: Classify(host)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Preference < out[j].Preference })
	return out
}

func normalizeHost(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
}

// isNotFound tells "this domain has no MX" from "the lookup failed". The first is a
// finding, the second is a gap, and they must not look the same.
func isNotFound(err error) bool {
	var dnsErr *net.DNSError
	if err == nil {
		return false
	}
	if errors.As(err, &dnsErr) {
		return dnsErr.IsNotFound
	}
	return false
}

func findSPF(records []string) string {
	for _, record := range records {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(record)), "v=spf1") {
			return strings.TrimSpace(record)
		}
	}
	return ""
}

func findDMARC(records []string) string {
	for _, record := range records {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(record)), "v=dmarc1") {
			return strings.TrimSpace(record)
		}
	}
	return ""
}

func spfIncludes(record string) []string {
	if record == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, field := range strings.Fields(record) {
		field = strings.TrimPrefix(field, "+")
		lower := strings.ToLower(field)
		var domain string
		switch {
		case strings.HasPrefix(lower, "include:"):
			domain = field[len("include:"):]
		case strings.HasPrefix(lower, "redirect="):
			domain = field[len("redirect="):]
		default:
			continue
		}
		domain = normalizeHost(domain)
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		out = append(out, domain)
	}
	return out
}

func classifyAll(domains []string) []Provider {
	var out []Provider
	seen := map[Provider]bool{}
	for _, domain := range domains {
		provider := Classify(domain)
		if provider == UnknownProvider || seen[provider] {
			continue
		}
		seen[provider] = true
		out = append(out, provider)
	}
	return out
}

func dmarcPolicy(record string) string {
	for _, field := range strings.Split(record, ";") {
		key, value, found := strings.Cut(strings.TrimSpace(field), "=")
		if found && strings.EqualFold(strings.TrimSpace(key), "p") {
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return ""
}

func (r Record) String() string {
	return fmt.Sprintf("%s: %s (%d MX)", r.Domain, r.Provider, len(r.MX))
}
