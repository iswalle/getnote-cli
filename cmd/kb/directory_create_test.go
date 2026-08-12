package kb

import "testing"

func TestResolveDirectoryCreateName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		args     []string
		nameFlag string
		want     string
		wantErr  bool
	}{
		{name: "位置参数", args: []string{"topic", "产品资料"}, want: "产品资料"},
		{name: "name 参数", args: []string{"topic"}, nameFlag: "产品资料", want: "产品资料"},
		{name: "缺少名称", args: []string{"topic"}, wantErr: true},
		{name: "重复指定", args: []string{"topic", "产品资料"}, nameFlag: "用户研究", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDirectoryCreateName(tt.args, tt.nameFlag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("name = %q, want %q", got, tt.want)
			}
		})
	}
}
