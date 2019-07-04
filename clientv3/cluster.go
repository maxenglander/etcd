// Copyright 2016 The etcd Authors
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

package clientv3

import (
	"context"
	"fmt"

	pb "go.etcd.io/etcd/etcdserver/etcdserverpb"
	"go.etcd.io/etcd/pkg/types"

	"google.golang.org/grpc"
)

type (
	Member                pb.Member
	MemberListResponse    pb.MemberListResponse
	MemberAddResponse     pb.MemberAddResponse
	MemberRemoveResponse  pb.MemberRemoveResponse
	MemberUpdateResponse  pb.MemberUpdateResponse
	MemberPromoteResponse pb.MemberPromoteResponse
)

type Cluster interface {
	// MemberList lists the current cluster membership.
	MemberList(ctx context.Context) (*MemberListResponse, error)

	// MemberAddAsAutoPromoting adds a new member as a learner that is
	// automatically promoted to a node upon catching up with the leader into the cluster.
	MemberAddAsAutoPromotingNode(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error)

	// MemberAddAsNode adds a new member as a node into the cluster.
	MemberAddAsNode(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error)

	// MemberAddAsLearner adds a new learner member into the cluster.
	MemberAddAsLearner(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error)

	// MemberRemove removes an existing member from the cluster.
	MemberRemove(ctx context.Context, id uint64) (*MemberRemoveResponse, error)

	// MemberUpdate updates the peer addresses of the member.
	MemberUpdate(ctx context.Context, id uint64, peerAddrs []string) (*MemberUpdateResponse, error)

	// MemberPromote promotes a member from raft learner (non-voting) to raft voting member.
	MemberPromote(ctx context.Context, id uint64) (*MemberPromoteResponse, error)
}

type cluster struct {
	remote   pb.ClusterClient
	callOpts []grpc.CallOption
}

func NewCluster(c *Client) Cluster {
	api := &cluster{remote: RetryClusterClient(c)}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func NewClusterFromClusterClient(remote pb.ClusterClient, c *Client) Cluster {
	api := &cluster{remote: remote}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (c *cluster) MemberAddAsAutoPromotingNode(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error) {
	return c.memberAdd(ctx, peerAddrs, true, true)
}

func (c *cluster) MemberAddAsNode(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error) {
	return c.memberAdd(ctx, peerAddrs, false, false)
}

func (c *cluster) MemberAddAsLearner(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error) {
	return c.memberAdd(ctx, peerAddrs, true, false)
}

func (c *cluster) memberAdd(ctx context.Context, peerAddrs []string, isLearner bool, autoPromote bool) (*MemberAddResponse, error) {
	// fail-fast before panic in rafthttp
	if _, err := types.NewURLs(peerAddrs); err != nil {
		return nil, err
	}

	r := &pb.MemberAddRequest{
		PeerURLs:    peerAddrs,
		IsLearner:   isLearner,
		AutoPromote: autoPromote,
	}
	fmt.Printf("Adding member isLearner=%t autoPromote=%t\n", isLearner, autoPromote)
	resp, err := c.remote.MemberAdd(ctx, r, c.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*MemberAddResponse)(resp), nil
}

func (c *cluster) MemberRemove(ctx context.Context, id uint64) (*MemberRemoveResponse, error) {
	r := &pb.MemberRemoveRequest{ID: id}
	resp, err := c.remote.MemberRemove(ctx, r, c.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*MemberRemoveResponse)(resp), nil
}

func (c *cluster) MemberUpdate(ctx context.Context, id uint64, peerAddrs []string) (*MemberUpdateResponse, error) {
	// fail-fast before panic in rafthttp
	if _, err := types.NewURLs(peerAddrs); err != nil {
		return nil, err
	}

	// it is safe to retry on update.
	r := &pb.MemberUpdateRequest{ID: id, PeerURLs: peerAddrs}
	resp, err := c.remote.MemberUpdate(ctx, r, c.callOpts...)
	if err == nil {
		return (*MemberUpdateResponse)(resp), nil
	}
	return nil, toErr(ctx, err)
}

func (c *cluster) MemberList(ctx context.Context) (*MemberListResponse, error) {
	// it is safe to retry on list.
	resp, err := c.remote.MemberList(ctx, &pb.MemberListRequest{}, c.callOpts...)
	if err == nil {
		return (*MemberListResponse)(resp), nil
	}
	return nil, toErr(ctx, err)
}

func (c *cluster) MemberPromote(ctx context.Context, id uint64) (*MemberPromoteResponse, error) {
	r := &pb.MemberPromoteRequest{ID: id}
	resp, err := c.remote.MemberPromote(ctx, r, c.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*MemberPromoteResponse)(resp), nil
}
