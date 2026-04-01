package server
type Limits struct{Tier string}
func LimitsFor(tier string)Limits{return Limits{Tier:tier}}
func(l Limits)IsPro()bool{return l.Tier=="pro"}
