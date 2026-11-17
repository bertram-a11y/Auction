package server

import (
	"context"
	"sync"
)	"fmt"

type Server struct {
	mu       sync.Mutex
	bidder   string
	highest  int
	isleader bool
	peers []pb.AuctionClient
	end time.Time

}

func (s *Server) Bid (con context.Context, request *pb.BidRequest) (*pb.BidReply, error) {
	if !s.Leader || request.Amount <= 0{
		return &pb.BidReply{Outcome: pb.BidReply_EXCEPTION} //Prøver næste node
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Now().After(s.end){
		return &pb.BidReply{Outcome: pb.BidReply_FAIL},nil 
	}
	if request.Amount <= s.highest {  //prøv igen efter et leader crash. Bid er allered igennem = succes
		if request.bidder == s.bidder && request.Amount == s.Amount {
			return &pb.BidReply{Outcome: pb.BidReply_SUCCES},nil 
		}
		return &pb.BidReply {Outcome: pb.BidReply_FAIL}
	}
	state := &pb.State

}

func (s *Server) Result(ctx conte) {

}

func (s *Server) Replicate {
	
}