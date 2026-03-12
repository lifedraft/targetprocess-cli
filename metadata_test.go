package targetprocess

import "testing"

const testMetaIndexXML = `<?xml version="1.0" encoding="utf-8"?>
<ResourceMetadataDescriptionIndex>
  <ResourceMetadataDescription Name="UserStory" Uri="/api/v1/UserStorys/meta" Description="Represents a user story" />
  <ResourceMetadataDescription Name="Bug" Uri="/api/v1/Bugs/meta" Description="Represents a bug" />
  <ResourceMetadataDescription Name="Task" Uri="/api/v1/Tasks/meta" Description="Represents a task" />
</ResourceMetadataDescriptionIndex>`

const testTypeMetaXML = `<?xml version="1.0" encoding="utf-8"?>
<ResourceMetadataDescription Name="UserStory">
  <ResourceMetadataPropertiesDescription>
    <ResourceMetadataPropertiesResourceValuesDescription>
      <ResourceFieldMetadataDescription Name="Id" Type="Int32" CanSet="false" CanGet="true" IsRequired="true" Description="Entity ID" />
      <ResourceFieldMetadataDescription Name="Name" Type="String" CanSet="true" CanGet="true" IsRequired="true" Description="Entity name" />
      <ResourceFieldMetadataDescription Name="Effort" Type="Decimal" CanSet="true" CanGet="true" IsRequired="false" Description="Total effort" />
    </ResourceMetadataPropertiesResourceValuesDescription>
    <ResourceMetadataPropertiesResourceReferencesDescription>
      <ResourceFieldMetadataDescription Name="Project" Type="Project" CanSet="true" CanGet="true" IsRequired="true" Description="Associated project" />
      <ResourceFieldMetadataDescription Name="EntityState" Type="EntityState" CanSet="true" CanGet="true" IsRequired="false" Description="Current state" />
    </ResourceMetadataPropertiesResourceReferencesDescription>
    <ResourceMetadataPropertiesResourceCollectionsDescription>
      <ResourceCollecitonFieldMetadataDescription Name="Tasks" Type="Task" CanSet="false" CanGet="true" IsRequired="false" Description="Child tasks" />
    </ResourceMetadataPropertiesResourceCollectionsDescription>
  </ResourceMetadataPropertiesDescription>
</ResourceMetadataDescription>`

func TestParseMetaIndex(t *testing.T) {
	types, err := parseMetaIndex([]byte(testMetaIndexXML))
	if err != nil {
		t.Fatalf("parseMetaIndex() error = %v", err)
	}
	if len(types) != 3 {
		t.Fatalf("got %d types, want 3", len(types))
	}

	tests := []struct {
		idx         int
		name        string
		description string
	}{
		{0, "UserStory", "Represents a user story"},
		{1, "Bug", "Represents a bug"},
		{2, "Task", "Represents a task"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := types[tt.idx]
			if ti.Name != tt.name {
				t.Errorf("Name = %q, want %q", ti.Name, tt.name)
			}
			if ti.Description != tt.description {
				t.Errorf("Description = %q, want %q", ti.Description, tt.description)
			}
		})
	}
}

func TestParseTypeMeta(t *testing.T) {
	fields, err := parseTypeMeta([]byte(testTypeMetaXML))
	if err != nil {
		t.Fatalf("parseTypeMeta() error = %v", err)
	}
	if len(fields) != 6 {
		t.Fatalf("got %d fields, want 6", len(fields))
	}

	// Check a value field.
	id := fields[0]
	if id.Name != "Id" {
		t.Errorf("field[0].Name = %q, want %q", id.Name, "Id")
	}
	if id.Kind != FieldKindValue {
		t.Errorf("field[0].Kind = %q, want %q", id.Kind, FieldKindValue)
	}
	if !id.Required {
		t.Error("field[0].Required = false, want true")
	}
	if !id.Readable {
		t.Error("field[0].Readable = false, want true")
	}
	if id.Writable {
		t.Error("field[0].Writable = true, want false")
	}

	// Check a reference field.
	project := fields[3]
	if project.Name != "Project" {
		t.Errorf("field[3].Name = %q, want %q", project.Name, "Project")
	}
	if project.Kind != FieldKindReference {
		t.Errorf("field[3].Kind = %q, want %q", project.Kind, FieldKindReference)
	}

	// Check a collection field.
	tasks := fields[5]
	if tasks.Name != "Tasks" {
		t.Errorf("field[5].Name = %q, want %q", tasks.Name, "Tasks")
	}
	if tasks.Kind != FieldKindCollection {
		t.Errorf("field[5].Kind = %q, want %q", tasks.Kind, FieldKindCollection)
	}
}
