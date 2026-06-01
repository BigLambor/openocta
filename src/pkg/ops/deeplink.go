package ops

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// BuildUIDeepLink returns an absolute UI URL for IM / webhook cards (P2-C1).
func BuildUIDeepLink(pathAndQuery string) string {
	base := strings.TrimSpace(os.Getenv("OPENOCTA_UI_BASE_URL"))
	if base == "" {
		return ""
	}
	p := strings.TrimPrefix(strings.TrimSpace(pathAndQuery), "/")
	return strings.TrimSuffix(base, "/") + "/" + p
}

// DomainFromInspectJobID maps cron job-inspect-* to navigation domain key.
func DomainFromInspectJobID(jobID string) string {
	switch strings.TrimPrefix(jobID, "job-inspect-") {
	case "hadoop":
		return DomainHadoop
	case "fi":
		return DomainFI
	case "gbase":
		return DomainGBase
	case "governance":
		return DomainGovernance
	case "dataapps":
		return DomainDataApps
	default:
		return ""
	}
}

// AlertGroupDeepLink builds UI link to alerts sub-tab for a group.
func AlertGroupDeepLink(domain, groupID string) string {
	if domain == "" || groupID == "" {
		return ""
	}
	q := url.Values{}
	q.Set("opsSubTab", "alerts")
	q.Set("alertGroup", groupID)
	return BuildUIDeepLink(fmt.Sprintf("%s?%s", domain, q.Encode()))
}
