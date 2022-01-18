package types

import (
	"time"
)

// Data structure as defined in https://github.com/bryanasdev000/microservice-jitsi-log .
type Jitsilog struct {
	Sala      string `json:"sala"`
	Curso     string `json:"curso"`
	Turma     string `json:"turma"`
	Aluno     string `json:"aluno"`
	Jid       string `json:"jid"`
	Email     string `json:"email"`
	Timestamp string `json:"timestamp"`
	time      time.Time
	Action    string `json:"action"`
}

func (jl *Jitsilog) GetTime() time.Time {
	return jl.time
}

func (jl *Jitsilog) SetTime(newTime time.Time) {
	jl.time = newTime
	jl.Timestamp = jl.time.Format(time.RFC3339)
}

func CabecalhoCSV() []string {
	return (&Jitsilog{
		Sala: "sala", Curso: "curso", Turma: "turma", Aluno: "aluno",
		Jid: "jid", Email: "email", Timestamp: "timestamp", Action: "action",
	}).RegistroCSV()
}

func (jl *Jitsilog) RegistroCSV() (r []string) {
	r = append(r, jl.Sala)
	r = append(r, jl.Curso)
	r = append(r, jl.Turma)
	r = append(r, jl.Aluno)
	r = append(r, jl.Jid)
	r = append(r, jl.Email)
	r = append(r, jl.Timestamp)
	r = append(r, jl.Action)
	return
}

type JitsilogSlice []*Jitsilog
type JitsilogIterator <-chan *Jitsilog
