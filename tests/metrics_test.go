package tests

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func downloadURLWithStatusCode(t *testing.T, u string) (int, string) {
	resp, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}

func subTestMetrics(t *testing.T, mc *mockServer) {
	mc.Do("SET", "metrics_test_1", "1", "FIELD", "foo", 5.5, "POINT", 5, 5)
	mc.Do("SET", "metrics_test_2", "2", "FIELD", "foo", 19.19, "POINT", 19, 19)
	mc.Do("SET", "metrics_test_2", "3", "FIELD", "foo", 19.19, "POINT", 19, 19)
	mc.Do("SET", "metrics_test_2", "truck1:driver", "STRING", "John Denton")

	status, index := downloadURLWithStatusCode(t, "http://127.0.0.1:4321/")
	if status != 200 {
		t.Fatalf("Expected status code 200, got: %d", status)
	}
	if !strings.Contains(index, "<a href") {
		t.Fatalf("missing link on index page")
	}

	status, metrics := downloadURLWithStatusCode(t, "http://127.0.0.1:4321/metrics")
	if status != 200 {
		t.Fatalf("Expected status code 200, got: %d", status)
	}
	for _, want := range []string{
		`tile38_connected_clients`,
		`tile38_cmd_duration_seconds_count{cmd="set"}`,
		`go_build_info`,
		`go_threads`,
		`tile38_collection_objects{col="metrics_test_1"} 1`,
		`tile38_collection_objects{col="metrics_test_2"} 3`,
		`tile38_collection_points{col="metrics_test_2"} 2`,
		`tile38_replication_info`,
		`role="leader"`,
	} {
		if !strings.Contains(metrics, want) {
			t.Fatalf("wanted metric: %s, got: %s", want, metrics)
		}
	}
}
