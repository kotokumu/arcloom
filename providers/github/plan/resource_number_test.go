package githubplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewResourceNumber(t *testing.T) {
	type args struct {
		value int64
	}
	tests := []struct {
		name    string
		args    args
		want    ResourceNumber
		wantErr bool
	}{
		{name: "smallest positive number", args: args{value: 1}, want: ResourceNumber{value: 1}},
		{name: "largest positive number", args: args{value: 9223372036854775807}, want: ResourceNumber{value: 9223372036854775807}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewResourceNumber(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewResourceNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ResourceNumber{})); diff != "" {
				t.Errorf("NewResourceNumber() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResourceNumber_Int64(t *testing.T) {
	type fields struct {
		value int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{name: "positive value", fields: fields{value: 42}, want: 42},
		{name: "zero value", fields: fields{value: 0}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := ResourceNumber{
				value: tt.fields.value,
			}
			got := n.Int64()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("ResourceNumber.Int64() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
