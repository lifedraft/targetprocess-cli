package targetprocess

import (
	"encoding/xml"
	"fmt"
)

// TypeInfo describes an entity type available in the Targetprocess API.
type TypeInfo struct {
	Name        string
	Description string
}

// FieldKind categorizes a field as a value, reference, or collection.
type FieldKind string

const (
	FieldKindValue      FieldKind = "value"
	FieldKindReference  FieldKind = "reference"
	FieldKindCollection FieldKind = "collection"
)

// FieldInfo describes a field/property of an entity type.
type FieldInfo struct {
	Name        string
	Type        string
	Kind        FieldKind
	Required    bool
	Readable    bool
	Writable    bool
	Description string
}

// XML structures for parsing TP metadata responses.

type xmlMetaIndex struct {
	XMLName xml.Name           `xml:"ResourceMetadataDescriptionIndex"`
	Types   []xmlMetaIndexItem `xml:"ResourceMetadataDescription"`
}

type xmlMetaIndexItem struct {
	Name        string `xml:"Name,attr"`
	Description string `xml:"Description,attr"`
}

type xmlTypeMeta struct {
	XMLName    xml.Name          `xml:"ResourceMetadataDescription"`
	Name       string            `xml:"Name,attr"`
	Properties xmlTypeProperties `xml:"ResourceMetadataPropertiesDescription"`
}

type xmlTypeProperties struct {
	Values      []xmlFieldMeta `xml:"ResourceMetadataPropertiesResourceValuesDescription>ResourceFieldMetadataDescription"`
	References  []xmlFieldMeta `xml:"ResourceMetadataPropertiesResourceReferencesDescription>ResourceFieldMetadataDescription"`
	Collections []xmlFieldMeta `xml:"ResourceMetadataPropertiesResourceCollectionsDescription>ResourceCollecitonFieldMetadataDescription"` //nolint:misspell // TP API has this typo in the XML schema
}

type xmlFieldMeta struct {
	Name        string `xml:"Name,attr"`
	Type        string `xml:"Type,attr"`
	CanSet      string `xml:"CanSet,attr"`
	CanGet      string `xml:"CanGet,attr"`
	IsRequired  string `xml:"IsRequired,attr"`
	Description string `xml:"Description,attr"`
}

func parseMetaIndex(data []byte) ([]TypeInfo, error) {
	var index xmlMetaIndex
	if err := xml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parsing metadata XML: %w", err)
	}

	types := make([]TypeInfo, len(index.Types))
	for i, t := range index.Types {
		types[i] = TypeInfo(t)
	}
	return types, nil
}

const xmlTrue = "true"

func parseTypeMeta(data []byte) ([]FieldInfo, error) {
	var meta xmlTypeMeta
	if err := xml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing type metadata XML: %w", err)
	}

	var fields []FieldInfo
	appendFields := func(raw []xmlFieldMeta, kind FieldKind) {
		for _, f := range raw {
			fields = append(fields, FieldInfo{
				Name:        f.Name,
				Type:        f.Type,
				Kind:        kind,
				Required:    f.IsRequired == xmlTrue,
				Readable:    f.CanGet == xmlTrue,
				Writable:    f.CanSet == xmlTrue,
				Description: f.Description,
			})
		}
	}
	appendFields(meta.Properties.Values, FieldKindValue)
	appendFields(meta.Properties.References, FieldKindReference)
	appendFields(meta.Properties.Collections, FieldKindCollection)

	return fields, nil
}
