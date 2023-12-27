package data

type Event struct {
	Id          uint64
	*Login      `json:"login,omitempty"`
	*Echo       `json:"echo,omitempty"`
	*TextTo     `json:"text_to,omitempty"`
	*TextFrom   `json:"text_from,omitempty"`
	*Disconnect `json:"disconnect,omitempty"`
	*Debug      `json:"debug,omitempty"`
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Echo struct {
	Text string `json:"text"`
}

type TextTo struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

type TextFrom struct {
	From string `json:"from"`
	Text string `json:"text"`
}

type Disconnect struct{}

type Debug struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}
