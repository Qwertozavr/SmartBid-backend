package config

import "testing"

func TestLoadDefaultsAdFinishedTopic(t *testing.T) {
	t.Setenv("KAFKA_AD_FINISHED_TOPIC", "")

	if got := Load().KafkaAdFinishedTopic; got != "ad-finished" {
		t.Fatalf("unexpected topic: %q", got)
	}
}
