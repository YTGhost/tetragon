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

	pb "github.com/cilium/cilium/api/v1/flow"
	v1 "github.com/cilium/tetragon/pkg/oldhubble/api/v1"
)

func filterByReplyField(replyParams []bool) FilterFunc {
	return func(ev *v1.Event) bool {
		if len(replyParams) == 0 {
			return true
		}
		switch ev.Event.(type) {
		case v1.Flow:
			reply := ev.GetFlow().GetReply()
			for _, replyParam := range replyParams {
				if reply == replyParam {
					return true
				}
			}
		}
		return false
	}
}

// ReplyFilter implements filtering for reply flows
type ReplyFilter struct{}

// OnBuildFilter builds a reply filter
func (r *ReplyFilter) OnBuildFilter(ctx context.Context, ff *pb.FlowFilter) ([]FilterFunc, error) {
	var fs []FilterFunc

	if ff.GetReply() != nil {
		fs = append(fs, filterByReplyField(ff.GetReply()))
	}

	return fs, nil
}
