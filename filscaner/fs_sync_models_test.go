package filscaner

import (
	"filscan_lotus/models"
	"filscan_lotus/utils"
	"testing"
)

func init() {
	models.Db_init(utils.GetConfiger())
}

func Test_fs_synced_tipset_path_models_upsert(t *testing.T) {
	type fields struct {
		Head *fs_synced_tipset
		Tail *fs_synced_tipset
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "insert", fields: fields{Head: &fs_synced_tipset{"tipset_key", "tipset_parent_key", 18}, Tail: &fs_synced_tipset{"tipset_key", "tipset_parent_key", 19}}, wantErr: false},
		{name: "insert", fields: fields{Head: &fs_synced_tipset{"tipset_key", "tipset_parent_key", 10}, Tail: &fs_synced_tipset{"tipset_key", "tipset_parent_key", 10}}, wantErr: false},
		{name: "insert", fields: fields{Head: &fs_synced_tipset{"tipset_key", "tipset_parent_key", 11}, Tail: &fs_synced_tipset{"tipset_key", "tipset_parent_key", 12}}, wantErr: false},
		{name: "update", fields: fields{Head: &fs_synced_tipset{"tipset_key_updated!!!!!", "tipset_parent_key_", 11}, Tail: &fs_synced_tipset{"tipset_key", "tipset_parent_key", 12}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := (&fs_synced_tipset_path{
				Head: tt.fields.Head,
				Tail: tt.fields.Tail,
			}).refresh_selc()
			if err := self.models_upsert(); (err != nil) != tt.wantErr {
				t.Errorf("models_upsert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_fs_synced_tipset_path_list_models_load(t *testing.T) {
	self := &fs_synced_tipset_path_list{}
	if err := self.models_load(); err != nil {
		t.Errorf("models_load() error = %v", err)
	}
}
