package classify

import (
	"context"
	"testing"
)

func TestClassifierRules(t *testing.T) {
	c := NewClassifier(nil, nil)
	ctx := context.Background()

	cases := []struct {
		name       string
		subject    string
		body       string
		wantSignal string
		wantMin    float64
	}{
		{
			name:       "PT Ack",
			subject:    "Recebemos sua candidatura para Desenvolvedor Go",
			body:       "Olá! Confirmamos que sua candidatura foi recebida com sucesso pela equipe.",
			wantSignal: SignalAck,
			wantMin:    0.85,
		},
		{
			name:       "EN Ack",
			subject:    "Thank you for applying to Stripe",
			body:       "We have received your application for Backend Engineer and our team will review it.",
			wantSignal: SignalAck,
			wantMin:    0.85,
		},
		{
			name:       "ES Ack",
			subject:    "Hemos recibido tu postulación",
			body:       "Gracias por postularte a la vacante de Ingeniero de Software.",
			wantSignal: SignalAck,
			wantMin:    0.85,
		},
		{
			name:       "PT Interview with Google Meet link",
			subject:    "Convite para entrevista técnica - Nubank",
			body:       "Gostaríamos de agendar um bate-papo técnico. Acesse https://meet.google.com/abc-defg-hij",
			wantSignal: SignalInterview,
			wantMin:    0.85,
		},
		{
			name:       "EN Interview with Calendly",
			subject:    "Invitation to interview at Figma",
			body:       "Please choose a time that works best for you on https://calendly.com/figma-recruiting/30min",
			wantSignal: SignalInterview,
			wantMin:    0.85,
		},
		{
			name:       "PT Assessment HackerRank",
			subject:    "Desafio técnico - Mercado Livre",
			body:       "Você foi aprovado para a etapa de teste técnico via HackerRank. Boa sorte!",
			wantSignal: SignalAssessment,
			wantMin:    0.85,
		},
		{
			name:       "EN Offer",
			subject:    "Offer Letter - Senior Software Engineer",
			body:       "We are pleased to offer you the position of Senior Software Engineer at Datadog.",
			wantSignal: SignalOffer,
			wantMin:    0.90,
		},
		{
			name:       "PT Rejection",
			subject:    "Atualização sobre o processo seletivo",
			body:       "Infelizmente decidimos não daremos continuidade à sua candidatura no momento. Desejamos sucesso em sua jornada.",
			wantSignal: SignalRejection,
			wantMin:    0.90,
		},
		{
			name:       "EN Rejection",
			subject:    "Your application at Apple",
			body:       "Unfortunately, we regret to inform you that we have decided to pursue other candidates.",
			wantSignal: SignalRejection,
			wantMin:    0.90,
		},
		{
			name:       "Other newsletter",
			subject:    "Weekly tech newsletter",
			body:       "Here are the top articles this week in tech.",
			wantSignal: SignalOther,
			wantMin:    0.20,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := c.Classify(ctx, tc.subject, tc.body)
			if res.Signal != tc.wantSignal {
				t.Errorf("Signal = %q, want %q", res.Signal, tc.wantSignal)
			}
			if res.Confidence < tc.wantMin {
				t.Errorf("Confidence = %f, want >= %f", res.Confidence, tc.wantMin)
			}
		})
	}
}
