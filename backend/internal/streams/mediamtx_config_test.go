package streams

import (
	"bytes"
	"strings"
	"testing"
)

func TestMediamtxConfigQuotesRTSPSourceWithAmpersand(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := configTmpl.Execute(&buf, configData{
		RecordDir: "/tmp/rec",
		Cameras: []cameraEntry{{
			ID:            "cam1",
			RTSPURL:       "rtsp://admin:secret@10.0.0.127:554/cam/realmonitor?channel=2&subtype=0",
			RecordEnabled: false,
		}},
		Profile: loadStreamProfile(),
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `source: "rtsp://`) {
		t.Fatalf("expected double-quoted source line, got:\n%s", out)
	}
	if strings.Contains(out, "source: rtsp://admin") {
		t.Fatalf("unquoted source leaks into config (YAML & can break parsing):\n%s", out)
	}
}
