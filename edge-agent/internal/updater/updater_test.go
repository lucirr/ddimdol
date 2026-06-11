package updater

import (
	"testing"
)

func TestRewriteToMirror(t *testing.T) {
	cases := []struct {
		name      string
		imageRef  string
		mirrorURL string
		want      string
	}{
		{
			name:      "central harbor host replaced",
			imageRef:  "harbor.central/edgedip/myapp:v1.2.3",
			mirrorURL: "https://harbor.edge",
			want:      "harbor.edge/edgedip/myapp:v1.2.3",
		},
		{
			name:      "mirror URL without scheme",
			imageRef:  "harbor.central/edgedip/myapp:v1.2.3",
			mirrorURL: "harbor.edge",
			want:      "harbor.edge/edgedip/myapp:v1.2.3",
		},
		{
			name:      "mirror URL with trailing slash",
			imageRef:  "harbor.central/edgedip/myapp:v1.2.3",
			mirrorURL: "https://harbor.edge/",
			want:      "harbor.edge/edgedip/myapp:v1.2.3",
		},
		{
			name:      "image with port in central host",
			imageRef:  "harbor.central:443/edgedip/myapp:v1.2.3",
			mirrorURL: "https://harbor.edge",
			want:      "harbor.edge/edgedip/myapp:v1.2.3",
		},
		{
			name:      "no registry host in imageRef",
			imageRef:  "edgedip/myapp:v1.2.3",
			mirrorURL: "https://harbor.edge",
			want:      "harbor.edge/edgedip/myapp:v1.2.3",
		},
		{
			name:      "bare image name",
			imageRef:  "ubuntu:22.04",
			mirrorURL: "https://harbor.edge",
			want:      "harbor.edge/ubuntu:22.04",
		},
		{
			name:      "mirror with http scheme",
			imageRef:  "harbor.central/edgedip/myapp:latest",
			mirrorURL: "http://harbor.edge",
			want:      "harbor.edge/edgedip/myapp:latest",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rewriteToMirror(tc.imageRef, tc.mirrorURL)
			if got != tc.want {
				t.Errorf("rewriteToMirror(%q, %q) = %q, want %q",
					tc.imageRef, tc.mirrorURL, got, tc.want)
			}
		})
	}
}
