import json
import jinja2
import logging

class ObjectDefinition:
    def __init__(self, name):
        self._atomic_fields = {}
        self._nonatomic_fields = {}
        self._array_fields = {}
        self._name = name

    def addAtomicField(self, name, type, omit=False):
        self._atomic_fields[name] = {
            "type": type,
            "omit": omit,
        }

    def addNonAtomicField(self, name, type, constraint, omit=False):
        self._nonatomic_fields[name] = {
            "type": type,
            "constraint": constraint,
            "omit": omit,
        }
    def addArrayField(self, name, type, constraint = [], omit=False, useValue=False):
        self._array_fields[name] = {
            "type": type,
            "constraint": constraint,
            "omit": omit,
            "useValue": useValue,
        }

    def __str__(self):
        template_str = """

const {{ Name }}Type = "{{ Name|lower }}"

type {{ Name }} struct {
        {% for field in AtomicFields %}
        {{ field|capitalize }} {{ AtomicFields[field]["type"] }} `json:"{{ '-' if AtomicFields[field]["omit"] else field|lower }}"`
        {%- endfor %}
        {% for field in NonAtomicFields %}
        {{ field|capitalize }} {{ NonAtomicFields[field]["type"] }} `json:"{{ '-' if NonAtomicFields[field]["omit"] else field|lower }}"`
        {%- endfor %}
        {% for field in ArrayFields %}
        {{ field|capitalize }} []{{ ArrayFields[field]["type"] }} `json:"{{ '-' if ArrayFields[field]["omit"] else field|lower }}"`
        {%- endfor %}
}

func (o *{{ Name }}) GetType() string {
    return {{ Name }}Type
}

func (o *{{ Name }}) MarshalJSON() (b []byte, e error) {
        type Copy {{ Name }}
    	return json.Marshal(&struct {
    		Type string `json:\"type\"`
    		*Copy
    	}{
    		Type: {{ Name }}Type,
    		Copy: (*Copy)(o),
    	})
}

func (o *{{ Name }}) UnmarshalJSON(b []byte) error {
    var objMap map[string]*json.RawMessage

    if err := json.Unmarshal(b, &objMap); err != nil {
        return err
    }

    {% for item in AtomicFields %}
    {% if not AtomicFields[item]["omit"] %}
    // TODO(jchaloup): check the objMap[\"{{ item|lower }}\"] actually exists
    if err := json.Unmarshal(*objMap[\"{{ item|lower }}\"], &o.{{ item|capitalize }}); err != nil {
        return err
    }
    {% endif %}
    {% endfor %}

    {% for item in NonAtomicFields %}
    // block for {{ item }} field
    {
        if objMap[\"{{ item|lower }}\"] != nil {
            var m map[string]interface{}
            if err := json.Unmarshal(*objMap[\"{{ item|lower }}\"], &m); err != nil {
                return err
            }

            switch dataType := m["type"]; dataType {
            {% for recursiveType in NonAtomicFields[item]["constraint"] %}
            case {{ recursiveType|capitalize }}Type:
                r := &{{ recursiveType|capitalize }}{}
                if err := json.Unmarshal(*objMap["{{ item|lower }}"], &r); err != nil {
                    return err
                }
                o.{{ item|capitalize }} = r
            {% endfor %}
            }
        }
    }
    {% endfor %}

    {% for item in ArrayFields %}
    // block for {{ item }} field
    {
        if objMap[\"{{ item|lower }}\"] != nil {
            var l []*json.RawMessage
            if err := json.Unmarshal(*objMap["{{ item|lower }}"], &l); err != nil {
                return err
            }

            o.{{ item }} = make([]{{ ArrayFields[item]["type"] }}, 0)
            for _, item := range l {
                var m map[string]interface{}
                if err := json.Unmarshal(*item, &m); err != nil {
                    return err
                }
                switch dataType := m["type"]; dataType {
                {% for recursiveType in ArrayFields[item]["constraint"] %}
                case {{ recursiveType }}Type:
                    r := &{{ recursiveType }}{}
                    if err := json.Unmarshal(*item, &r); err != nil {
                        return err
                    }
                    {% if ArrayFields[item]["useValue"] -%}
                    o.{{ item }} = append(o.{{ item }}, *r)
                    {% else -%}
                    o.{{ item }} = append(o.{{ item }}, r)
                    {% endif %}
                {% endfor %}
                }
            }
        }
    }
    {% endfor %}

    return nil
}

"""

        template_vars = {
            "Name": self._name,
            "AtomicFields": self._atomic_fields,
            "NonAtomicFields": self._nonatomic_fields,
            "ArrayFields": self._array_fields,
        }

        return jinja2.Environment().from_string(template_str).render(template_vars)

def getConstraints(items):
    constraints = []
    for recType in items:
        if "$ref" not in recType:
            logging.error("Item %s is not '$ref'" % (recType))
            continue
        c = recType["$ref"].split("/")[-1].capitalize()
        #if c not in ["struct", "identifier", "channel", "slice"]:
        #    continue
        constraints.append( c )
    return constraints

class DataTypeGenerator(object):
    def __init__(self, definitions, dataTypes):
        self.definitions = definitions
        self.output = ""
        self.dataTypes = dataTypes

    def parse(self):
        self.output = ""

        for definition in self.definitions:
            if definition not in self.dataTypes:
                continue

            self.parseDefinition(definition.capitalize(), self.definitions[definition])

        return self

    def parseDefinition(self, dataType, definition):
        #print dataType
        #print definition

        obj = ObjectDefinition(dataType)

        if definition["type"] == "object":
            for property in definition["properties"]:
                itemType = definition["properties"][property]["type"]
                # atomic type
                if itemType == "string":
                    # skip all 'type' fields
                    if property == "type":
                        continue
                    if definition["properties"][property]["description"] == "!!omit":
                        obj.addAtomicField(property.capitalize(), itemType, omit=True)
                    else:
                        obj.addAtomicField(property.capitalize(), itemType, omit=False)
                elif itemType == "boolean":
                    # skip all 'type' fields
                    if property == "type":
                        continue
                    if definition["properties"][property]["description"] == "!!omit":
                        obj.addAtomicField(property.capitalize(), "bool", omit=True)
                    else:
                        obj.addAtomicField(property.capitalize(), "bool", omit=False)
                # list of permited types
                elif itemType == "object":
                    ok = False
                    for keyOf in ["oneOf", "anyOf"]:
                        if keyOf in definition["properties"][property]:
                            constraints = getConstraints(definition["properties"][property][keyOf])
                            obj.addNonAtomicField(property.capitalize(), "DataType", constraints)
                            ok = True
                            break
                    if not ok:
                        logging.error("No anyOf nor OneOf iproperty.capitalize()n: %s" % items)
                        exit(1)
                elif itemType == "array":
                    items = definition["properties"][property]["items"]
                    if items["type"] == "object" and "properties" not in items:
                        ok = False
                        for keyOf in ["oneOf", "anyOf"]:
                            if keyOf in items:
                                constraints = getConstraints(items[keyOf])
                                obj.addArrayField(property.capitalize(), "DataType", constraints )
                                ok = True
                                break
                        if not ok:
                            logging.error("No anyOf nor OneOf in: %s" % items)
                            exit(1)
                    else:
                        # parse items first
                        itemDef = self.parseDefinition("%s%sItem" % (dataType.capitalize(), property.capitalize()), definition["properties"][property]["items"])
                        obj.addArrayField(property.capitalize(), "%s%sItem" % (dataType.capitalize(), property.capitalize()), ["%s%sItem" % (dataType.capitalize(), property.capitalize())], useValue=True)
                else:
                    logging.error("Unrecognized type: %s" % itemType)
                    exit(1)

        self.output += str(obj)
        return obj

    def __str__(self):
            return """package types

        import "encoding/json"

        // DataType is
        type DataType interface {
        	GetType() string
        }""" + self.output

def printSymbolDefinition(dataTypes):
        template_str = """

package symbols

import (
    "encoding/json"
    gotypes "github.com/gofed/symbols-extractor/pkg/types"
)

type SymbolDef struct {
       Pos     string   `json:"pos"`
       Name    string   `json:"name"`
       Package string   `json:"package"`
       Def     gotypes.DataType `json:"def"`
       Block   int              `json:"block"`
}

func (o *SymbolDef) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage

	if err := json.Unmarshal(b, &objMap); err != nil {
		return err
	}

	if err := json.Unmarshal(*objMap["pos"], &o.Pos); err != nil {
		return err
	}

	if err := json.Unmarshal(*objMap["name"], &o.Name); err != nil {
		return err
	}

    if err := json.Unmarshal(*objMap["package"], &o.Package); err != nil {
		return err
	}

    if _, ok := objMap["block"]; ok {
		if err := json.Unmarshal(*objMap["block"], &o.Block); err != nil {
			return err
		}
    }

	var m map[string]interface{}
	if err := json.Unmarshal(*objMap["def"], &m); err != nil {
		return err
	}

    switch dataType := m["type"]; dataType {
    {% for recursiveType in DataTypes %}
    case gotypes.{{ recursiveType|capitalize }}Type:
        r := &gotypes.{{ recursiveType|capitalize }}{}
        if err := json.Unmarshal(*objMap["def"], &r); err != nil {
            return err
        }
        o.Def = r
    {% endfor %}
    case gotypes.NilType:
		r := &gotypes.Nil{}
		if err := json.Unmarshal(*objMap["def"], &r); err != nil {
			return err
		}
		o.Def = r
    }

	return nil
}"""

        template_vars = {
            "DataTypes": dataTypes,
        }

        return jinja2.Environment().from_string(template_str).render(template_vars)


if __name__ == "__main__":
    with open("golang-project-exported-api.json", "r") as f:
        data = json.load(f)

    dataTypes = ["identifier", "builtin", "constant", "packagequalifier", "selector", "channel", "slice", "array", "map", "pointer", "ellipsis", "function", "method", "interface", "struct"]

    with open("pkg/types/types.go", "w") as file:
        file.write(str(DataTypeGenerator(data["definitions"], dataTypes).parse()))

    with open("pkg/symbols/symbol.go", "w") as file:
        file.write(printSymbolDefinition(dataTypes))
