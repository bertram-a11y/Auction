package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	pb "auction/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	auctionDuration = 100 * time.Second
	rpcTimeout      = 1 * time.Second
	pingInterval    = 500 * time.Millisecond
	startupGrace    = 5 * time.Second
)

type Server struct {
	pb.UnimplementedAuctionServer

	mu       sync.Mutex
	id       int
	leaderID int
	peers    map[int]pb.AuctionClient
	alive    map[int]bool

	bidder  string
	highest int64
	end     time.Time
}

func (s *Server) Bid(ctx context.Context, req *pb.BidRequest) (*pb.BidReply, error) {
	if req == nil || req.GetBidder() == "" || req.GetAmount() <= 0 {
		return &pb.BidReply{Outcome: pb.BidReply_EXCEPTION}, nil
	}

	s.mu.Lock()
	if s.leaderID != s.id {
		leader, ok := s.peers[s.leaderID]
		s.mu.Unlock()
		if !ok {
			return &pb.BidReply{Outcome: pb.BidReply_EXCEPTION}, nil
		}
		fctx, cancel := context.WithTimeout(ctx, 2*rpcTimeout)
		defer cancel()
		reply, err := leader.Bid(fctx, req)
		if err != nil {
			return &pb.BidReply{Outcome: pb.BidReply_EXCEPTION}, nil
		}
		return reply, nil
	}
	defer s.mu.Unlock()

	if !time.Now().Before(s.end) {
		return &pb.BidReply{Outcome: pb.BidReply_FAIL}, nil
	}
	if req.GetAmount() <= s.highest {
		if req.GetAmount() == s.highest && req.GetBidder() == s.bidder {
			return &pb.BidReply{Outcome: pb.BidReply_SUCCES}, nil
		}
		return &pb.BidReply{Outcome: pb.BidReply_FAIL}, nil
	}

	state := &pb.State{
		HighestBidder: req.GetBidder(),
		Highest:       req.GetAmount(),
		EndUnix:       s.end.Unix(),
	}

	for id, peer := range s.peers {
		if !s.alive[id] {
			continue
		}
		rctx, cancel := context.WithTimeout(ctx, rpcTimeout)
		_, err := peer.Replicate(rctx, state)
		cancel()
		if err != nil {
			log.Printf("node %d: node %d is not responding, marking as dead", s.id, id)
			s.alive[id] = false
		}
	}

	s.bidder = state.GetHighestBidder()
	s.highest = state.GetHighest()
	return &pb.BidReply{Outcome: pb.BidReply_SUCCES}, nil
}

func (s *Server) Result(ctx context.Context, _ *pb.Empty) (*pb.ResultReply, error) {
	s.mu.Lock()
	if s.leaderID != s.id {
		leader, ok := s.peers[s.leaderID]
		s.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("no leader found")
		}
		fctx, cancel := context.WithTimeout(ctx, rpcTimeout)
		defer cancel()
		return leader.Result(fctx, &pb.Empty{})
	}
	defer s.mu.Unlock()

	return &pb.ResultReply{
		Over:   !time.Now().Before(s.end),
		Winner: s.bidder,
		Amount: s.highest,
	}, nil
}

func (s *Server) Replicate(_ context.Context, state *pb.State) (*pb.Empty, error) {
	if state == nil || state.GetHighest() < 0 {
		return nil, fmt.Errorf("invalid auction state")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if state.GetHighest() > s.highest {
		s.bidder = state.GetHighestBidder()
		s.highest = state.GetHighest()
		s.end = time.Unix(state.GetEndUnix(), 0)
	}
	return &pb.Empty{}, nil
}

func (s *Server) Ping(_ context.Context, _ *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

func (s *Server) electLeader() {
	leader := s.id
	for id, ok := range s.alive {
		if ok && id < leader {
			leader = id
		}
	}
	if leader != s.leaderID {
		log.Printf("node %d: leader is now node %d", s.id, leader)
	}
	s.leaderID = leader
}

func (s *Server) monitor() {
	time.Sleep(startupGrace)
	for {
		time.Sleep(pingInterval)

		results := map[int]bool{}
		for id, p := range s.peers {
			ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
			_, err := p.Ping(ctx, &pb.Empty{})
			cancel()
			results[id] = err == nil
		}

		s.mu.Lock()
		for id, ok := range results {
			if s.alive[id] && !ok {
				log.Printf("node %d: node %d has crashed", s.id, id)
			}
			s.alive[id] = s.alive[id] && ok
		}
		s.electLeader()
		s.mu.Unlock()
	}
}

func main() {
	id := flag.Int("id", 1, "id of this node (1, 2, 3 ...)")
	nodes := flag.String("nodes", "5001,5002,5003", "ports of all nodes; node 1 = first port")
	flag.Parse()

	ports := strings.Split(*nodes, ",")
	if *id < 1 || *id > len(ports) {
		log.Fatalf("id must be between 1 and %d", len(ports))
	}

	s := &Server{
		id:    *id,
		peers: map[int]pb.AuctionClient{},
		alive: map[int]bool{},
		end:   time.Now().Add(auctionDuration),
	}
	for i, p := range ports {
		pid := i + 1
		if pid == *id {
			continue
		}
		conn, err := grpc.NewClient("localhost:"+p, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatal(err)
		}
		s.peers[pid] = pb.NewAuctionClient(conn)
		s.alive[pid] = true
	}
	s.electLeader()

	lis, err := net.Listen("tcp", ":"+ports[*id-1])
	if err != nil {
		log.Fatal(err)
	}
	g := grpc.NewServer()
	pb.RegisterAuctionServer(g, s)

	go s.monitor()
	log.Printf("node %d listening on port %s, leader is node %d", *id, ports[*id-1], s.leaderID)
	log.Fatal(g.Serve(lis))
}
