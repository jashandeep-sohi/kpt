package location

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

var (
	testReader  = &bytes.Buffer{}
	testWriter  = &bytes.Buffer{}
	testOptions = []Option{WithOci(), WithGit(), WithDir()}
)

//nolint:scopelint
func TestParseReference(t *testing.T) {
	type args struct {
		location string
		opts     []Option
	}
	tests := []struct {
		name    string
		args    args
		want    Reference
		wantErr bool
	}{
		{
			name: "OciSimpleName",
			args: args{
				location: "oci://ubuntu",
			},
			want: Oci{
				Image:     name.MustParseReference("ubuntu"),
				Directory: ".",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithTag",
			args: args{
				location: "oci://my-registry.local/name:tag",
			},
			want: Oci{
				Image:     name.MustParseReference("my-registry.local/name:tag"),
				Directory: ".",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithDigest",
			args: args{
				location: "oci://my-registry.local/name@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba",
			},
			want: Oci{
				Image:     name.MustParseReference("my-registry.local/name@sha256:9f6ca9562c5e7bd8bb53d736a2869adc27529eb202996dfefb804ec2c95237ba"),
				Directory: ".",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithDirectory",
			args: args{
				location: "oci://my-registry.local/name//sub/directory",
			},
			want: Oci{
				Image:     name.MustParseReference("my-registry.local/name:latest"),
				Directory: "sub/directory",
			},
			wantErr: false,
		},
		{
			name: "OciNameWithDirectoryAndTag",
			args: args{
				location: "oci://my-registry.local/name//sub/directory:tag",
			},
			want: Oci{
				Image:     name.MustParseReference("my-registry.local/name:tag"),
				Directory: "sub/directory",
			},
			wantErr: false,
		},
		{
			name: "OciDirectoryExtraSlashes",
			args: args{
				location: "oci://my-registry.local/name///sub/directory",
			},
			want: Oci{
				Image:     name.MustParseReference("my-registry.local/name:latest"),
				Directory: "sub/directory",
			},
			wantErr: false,
		},
		{
			name: "OciDirectoryEmptyPath",
			args: args{
				location: "oci://my-registry.local/name//",
			},
			want: Oci{
				Image:     name.MustParseReference("my-registry.local/name:latest"),
				Directory: ".",
			},
			wantErr: false,
		},
		{
			name: "GitSimpleRepo",
			args: args{
				location: "https://hostname/repo.git@main",
			},
			want: Git{
				Repo:      "https://hostname/repo",
				Directory: ".",
				Ref:       "main",
			},
			wantErr: false,
		},
		{
			name: "GitWithDirectory",
			args: args{
				location: "https://hostname/repo.git/sub/directory@main",
			},
			want: Git{
				Repo:      "https://hostname/repo",
				Directory: "sub/directory",
				Ref:       "main",
			},
			wantErr: false,
		},
		{
			name: "SimpleDir",
			args: args{
				location: "path/to/directory",
			},
			want: Dir{
				Directory: "path/to/directory",
			},
			wantErr: false,
		},
		{
			name: "InputStream",
			args: args{
				location: "-",
				opts:     []Option{WithStdin(testReader)},
			},
			want: InputStream{
				Reader: testReader,
			},
			wantErr: false,
		},
		{
			name: "OutputStream",
			args: args{
				location: "-",
				opts:     []Option{WithStdout(testWriter)},
			},
			want: OutputStream{
				Writer: testWriter,
			},
			wantErr: false,
		},
		{
			name: "StreamLocationNotExpected",
			args: args{
				location: "-",
			},
			want: Dir{
				Directory: "-",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReference(tt.args.location, append(tt.args.opts, testOptions...)...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}