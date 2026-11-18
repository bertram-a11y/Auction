package server

import (
	"context"
	"sync"
	"time"

	pb "auction/proto"
)

type Server struct {
	pb.UnimplementedAuctionServer

	mu       sync.Mutex
	bidder   string
	highest  int64
	isLeader bool
	peers    []pb.AuctionClient
	end      time.Time
}

func (s *Server) Bid(con context.Context, request *pb.BidRequest) (*pb.BidReply, error) {
	if !s.isLeader || request.Amount <= 0 {
		return &pb.BidReply{Outcome: pb.BidReply_EXCEPTION}, nil //Prøver næste node
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Now().After(s.end) {
		return &pb.BidReply{Outcome: pb.BidReply_FAIL}, nil
	}
	if request.Amount <= s.highest { //prøv igen efter et leader crash. Bid er allered igennem = succes
		if request.GetBidder() == s.bidder && request.Amount == s.highest {
			return &pb.BidReply{Outcome: pb.BidReply_SUCCES}, nil
		}
		return &pb.BidReply{Outcome: pb.BidReply_FAIL}, nil
	}

}

func (s *Server) Result(ctx context.Context) {

}

func (s *Server) Replicate() {

}
