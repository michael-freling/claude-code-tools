package kube

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// RenderOptions holds options for generating RBAC manifests.
type RenderOptions struct {
	ClusterRoleName         string
	ServiceAccountName      string
	ServiceAccountNamespace string
	Kubeconfig              string
	Context                 string
}

// APIResource represents a discovered Kubernetes API resource.
type APIResource struct {
	APIGroup   string
	Resource   string
	Namespaced bool
	Verbs      []string
}

// Render discovers API resources from the cluster and generates RBAC YAML
// manifests (ServiceAccount, ClusterRole, ClusterRoleBinding) applying the
// carveout rules.
func Render(opts RenderOptions) (string, error) {
	resources, err := discoverResources(opts.Kubeconfig, opts.Context)
	if err != nil {
		return "", fmt.Errorf("failed to discover API resources: %w", err)
	}

	rules := buildRules(resources)
	return renderYAML(opts, rules), nil
}

// RenderFromResources generates RBAC YAML from pre-discovered resources.
// Useful for testing without a live cluster.
func RenderFromResources(opts RenderOptions, resources []APIResource) string {
	rules := buildRules(resources)
	return renderYAML(opts, rules)
}

// PolicyRule represents a single RBAC policy rule.
type PolicyRule struct {
	APIGroups []string
	Resources []string
	Verbs     []string
}

func buildRules(resources []APIResource) []PolicyRule {
	// Group resources by (apiGroup, namespaced/cluster-scoped)
	namespacedByGroup := make(map[string][]string)
	clusterByGroup := make(map[string][]string)

	for _, r := range resources {
		if IsAPIGroupDenied(r.APIGroup) {
			continue
		}
		// Check core-group carveouts
		if r.APIGroup == "" {
			if IsResourceDenied(r.Resource) {
				continue
			}
			if IsSubresourceDenied(r.Resource) {
				continue
			}
		}
		// Check subresource carveouts for any group
		if strings.Contains(r.Resource, "/") && IsSubresourceDenied(r.Resource) {
			continue
		}

		if r.Namespaced {
			namespacedByGroup[r.APIGroup] = append(namespacedByGroup[r.APIGroup], r.Resource)
		} else {
			clusterByGroup[r.APIGroup] = append(clusterByGroup[r.APIGroup], r.Resource)
		}
	}

	var rules []PolicyRule

	// Emit namespaced rules with full verbs
	groups := sortedKeys(namespacedByGroup)
	for _, group := range groups {
		resources := namespacedByGroup[group]
		sort.Strings(resources)
		rules = append(rules, PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     FilterVerbs([]string{"*"}),
		})
	}

	// Emit cluster-scoped rules with read-only verbs
	groups = sortedKeys(clusterByGroup)
	for _, group := range groups {
		resources := clusterByGroup[group]
		sort.Strings(resources)
		rules = append(rules, PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     ReadOnlyVerbs(),
		})
	}

	return rules
}

func renderYAML(opts RenderOptions, rules []PolicyRule) string {
	var buf bytes.Buffer

	// ServiceAccount
	fmt.Fprintf(&buf, `apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
---
`, opts.ServiceAccountName, opts.ServiceAccountNamespace)

	// ClusterRole
	fmt.Fprintf(&buf, `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s
rules:
`, opts.ClusterRoleName)

	for _, rule := range rules {
		fmt.Fprintf(&buf, "- apiGroups:\n")
		for _, g := range rule.APIGroups {
			if g == "" {
				fmt.Fprintf(&buf, "  - \"\"\n")
			} else {
				fmt.Fprintf(&buf, "  - %s\n", g)
			}
		}
		fmt.Fprintf(&buf, "  resources:\n")
		for _, r := range rule.Resources {
			fmt.Fprintf(&buf, "  - %s\n", r)
		}
		fmt.Fprintf(&buf, "  verbs:\n")
		for _, v := range rule.Verbs {
			fmt.Fprintf(&buf, "  - %s\n", v)
		}
	}

	// ClusterRoleBinding
	fmt.Fprintf(&buf, `---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s
subjects:
- kind: ServiceAccount
  name: %s
  namespace: %s
`, opts.ClusterRoleName, opts.ClusterRoleName, opts.ServiceAccountName, opts.ServiceAccountNamespace)

	return buf.String()
}

// discoverResources calls kubectl api-resources to discover available resources.
var discoverResources = func(kubeconfig, context string) ([]APIResource, error) {
	args := []string{"api-resources", "-o", "wide", "--no-headers"}
	if kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	if context != "" {
		args = append(args, "--context", context)
	}

	out, err := exec.Command("kubectl", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl api-resources failed: %w", err)
	}

	return parseAPIResources(string(out))
}

// parseAPIResources parses the output of `kubectl api-resources -o wide --no-headers`.
func parseAPIResources(output string) ([]APIResource, error) {
	var resources []APIResource

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// kubectl api-resources -o wide --no-headers output format:
		// NAME SHORTNAMES APIVERSION NAMESPACED KIND VERBS
		// Fields are whitespace-separated but SHORTNAMES can be empty, so we
		// need to handle variable column counts.
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		name := fields[0]

		// Find NAMESPACED column (true/false)
		namespacedIdx := -1
		for i, f := range fields {
			if f == "true" || f == "false" {
				namespacedIdx = i
				break
			}
		}
		if namespacedIdx < 0 || namespacedIdx < 2 {
			continue
		}

		// APIVERSION is the field before NAMESPACED
		apiVersion := fields[namespacedIdx-1]
		namespaced := fields[namespacedIdx] == "true"

		// Parse apiGroup from APIVERSION (group/version or just version for core)
		apiGroup := ""
		if parts := strings.Split(apiVersion, "/"); len(parts) == 2 {
			apiGroup = parts[0]
		}

		// Parse VERBS from last field (in square brackets)
		var verbs []string
		lastField := fields[len(fields)-1]
		if strings.HasPrefix(lastField, "[") {
			// Collect all bracket-enclosed verb fields
			verbStr := ""
			for i := len(fields) - 1; i >= 0; i-- {
				if strings.HasPrefix(fields[i], "[") {
					verbStr = strings.Join(fields[i:], " ")
					break
				}
			}
			verbStr = strings.Trim(verbStr, "[]")
			verbs = append(verbs, strings.Fields(verbStr)...)
		}

		resources = append(resources, APIResource{
			APIGroup:   apiGroup,
			Resource:   name,
			Namespaced: namespaced,
			Verbs:      verbs,
		})
	}

	return resources, nil
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
