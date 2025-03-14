// Copyright 2019-2020 Authors of Hubble
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filters

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	pb "github.com/cilium/cilium/api/v1/flow"
	v1 "github.com/cilium/tetragon/pkg/oldhubble/api/v1"
)

func sourceFQDN(ev *v1.Event) []string {
	return ev.GetFlow().GetSourceNames()
}

func destinationFQDN(ev *v1.Event) []string {
	return ev.GetFlow().GetDestinationNames()
}

var (
	fqdnFilterAllowedChars  = "[-a-zA-Z0-9_.]*"
	fqdnFilterIsValidFilter = regexp.MustCompile("^[-a-zA-Z0-9_.*]+$")
)

func parseFQDNFilter(pattern string) (*regexp.Regexp, error) {
	pattern = strings.ToLower(pattern)
	pattern = strings.TrimSpace(pattern)
	pattern = strings.TrimSuffix(pattern, ".")

	if !fqdnFilterIsValidFilter.MatchString(pattern) {
		return nil, fmt.Errorf(`only alphanumeric ASCII characters, the hyphen "-", "." and "*" are allowed: %s`,
			pattern)
	}

	// "." becomes a literal .
	pattern = strings.Replace(pattern, ".", "[.]", -1)

	// "*" becomes a zero or more of the allowed characters
	pattern = strings.Replace(pattern, "*", fqdnFilterAllowedChars, -1)

	return regexp.Compile("^" + pattern + "$")
}

func filterByFQDNs(fqdnPatterns []string, getFQDNs func(*v1.Event) []string) (FilterFunc, error) {
	matchPatterns := make([]*regexp.Regexp, 0, len(fqdnPatterns))
	for _, pattern := range fqdnPatterns {
		re, err := parseFQDNFilter(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid FQDN in filter: %s", err)
		}
		matchPatterns = append(matchPatterns, re)
	}

	return func(ev *v1.Event) bool {
		names := getFQDNs(ev)
		if len(names) == 0 {
			return false
		}

		for _, name := range names {
			for _, re := range matchPatterns {
				if re.MatchString(name) {
					return true
				}
			}
		}

		return false
	}, nil
}

// filterByDNSQueries returns a FilterFunc that filters a flow by L7.DNS.query field.
// The filter function returns true if and only if the DNS query field matches any of
// the regular expressions.
func filterByDNSQueries(queryPatterns []string) (FilterFunc, error) {
	var queries []*regexp.Regexp
	for _, pattern := range queryPatterns {
		query, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile regexp: %v", err)
		}
		queries = append(queries, query)
	}
	return func(ev *v1.Event) bool {
		dns := ev.GetFlow().GetL7().GetDns()
		if dns == nil {
			return false
		}
		for _, query := range queries {
			if query.MatchString(dns.Query) {
				return true
			}
		}
		return false
	}, nil
}

// FQDNFilter implements filtering based on FQDN information
type FQDNFilter struct{}

// OnBuildFilter builds a FQDN filter
func (f *FQDNFilter) OnBuildFilter(ctx context.Context, ff *pb.FlowFilter) ([]FilterFunc, error) {
	var fs []FilterFunc

	if ff.GetSourceFqdn() != nil {
		ff, err := filterByFQDNs(ff.GetSourceFqdn(), sourceFQDN)
		if err != nil {
			return nil, err
		}
		fs = append(fs, ff)
	}

	if ff.GetDestinationFqdn() != nil {
		ff, err := filterByFQDNs(ff.GetDestinationFqdn(), destinationFQDN)
		if err != nil {
			return nil, err
		}
		fs = append(fs, ff)
	}

	if ff.GetDnsQuery() != nil {
		dnsFilters, err := filterByDNSQueries(ff.GetDnsQuery())
		if err != nil {
			return nil, fmt.Errorf("invalid DNS query filter: %v", err)
		}
		fs = append(fs, dnsFilters)
	}

	return fs, nil
}
