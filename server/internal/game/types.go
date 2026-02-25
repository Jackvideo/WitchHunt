package game

type Phase string

const (
	PhaseDay          Phase = "day"
	PhaseNightWitch   Phase = "night_witch"
	PhaseNightSheriff Phase = "night_sheriff"
	PhaseNightResult  Phase = "night_result"
	PhaseTrial        Phase = "trial"
	PhaseGameOver     Phase = "game_over"
)

type IdentityType string

const (
	IDVillager IdentityType = "villager"
	IDWitch    IdentityType = "witch"
	IDSheriff  IdentityType = "sheriff"
)

type CardColor string

const (
	ColorRed   CardColor = "red"
	ColorGreen CardColor = "green"
	ColorBlue  CardColor = "blue"
	ColorBlack CardColor = "black"
)

type CardType string

const (
	CTAccuse1   CardType = "accuse_1"
	CTAccuse2   CardType = "accuse_2"
	CTAccuse3   CardType = "accuse_3"
	CTNight     CardType = "night"
	CTContagion CardType = "contagion"
	CTBlackCat  CardType = "black_cat"
	CTSanctuary CardType = "sanctuary"
	CTDevotee   CardType = "devotee"
	CTFrame     CardType = "frame"
	CTArson     CardType = "arson"
	CTDetention CardType = "detention"
	CTDefense   CardType = "defense"
	CTRobbery   CardType = "robbery"
	CTCurse     CardType = "curse"
)

var CardNames = map[CardType]string{
	CTAccuse1: "指控(1)", CTAccuse2: "指控(2)", CTAccuse3: "指控(3)",
	CTNight: "夜晚", CTContagion: "传染",
	CTBlackCat: "黑猫", CTSanctuary: "避难所", CTDevotee: "信徒",
	CTFrame: "嫁祸", CTArson: "纵火", CTDetention: "拘留",
	CTDefense: "辩护", CTRobbery: "抢劫", CTCurse: "诅咒",
}

var IdentityNames = map[IdentityType]string{
	IDVillager: "村民", IDWitch: "女巫", IDSheriff: "警长",
}

type IdentityCard struct {
	Type     IdentityType `json:"type"`
	Revealed bool         `json:"revealed"`
}

type Card struct {
	ID    string    `json:"id"`
	Type  CardType  `json:"type"`
	Color CardColor `json:"color"`
	Value int       `json:"value,omitempty"`
}

type Player struct {
	UserID      uint            `json:"user_id"`
	Username    string          `json:"username"`
	Identities  []*IdentityCard `json:"-"`
	Hand        []*Card         `json:"-"`
	Equipment   []*Card         `json:"equipment"`
	Accusations []*Card         `json:"accusations"`
	AccuseTotal int             `json:"accuse_total"`
	Alive       bool            `json:"alive"`
	IsWitch     bool            `json:"-"`
	Detained    int             `json:"detained"`
	HasHammer   bool            `json:"has_hammer"`
}

func (p *Player) HasEquipment(t CardType) bool {
	for _, c := range p.Equipment {
		if c.Type == t {
			return true
		}
	}
	return false
}

func (p *Player) RemoveHandCard(id string) *Card {
	for i, c := range p.Hand {
		if c.ID == id {
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			return c
		}
	}
	return nil
}

func (p *Player) Unrevealed() []*IdentityCard {
	var out []*IdentityCard
	for _, id := range p.Identities {
		if !id.Revealed {
			out = append(out, id)
		}
	}
	return out
}

type PlayerInfo struct {
	UserID   uint
	Username string
}

type Action struct {
	Type          string `json:"type"`
	CardID        string `json:"card_id,omitempty"`
	TargetID      uint   `json:"target_id,omitempty"`
	ExtraTargetID uint   `json:"extra_target_id,omitempty"`
	CardIndex     int    `json:"card_index"`
}

type Event struct {
	Message string `json:"message"`
}

type ActionResult struct {
	Events []Event `json:"events"`
}

type TrialInfo struct {
	AccusedID uint `json:"accused_id"`
	FlipperID uint `json:"flipper_id"`
}
