package g

import (
	"strings"

	"github.com/integrii/flaggy"
	"github.com/spf13/cast"
)

// CliArr is the array of CliSC
var CliArr = []*CliSC{}

// CliSC represents a CLI subcommand
type CliSC struct {
	Name        string
	Description string
	Singular    string
	Sc          *flaggy.Subcommand
	Vals        map[string]interface{}
	PosFlags    []Flag
	Flags       []Flag
	CrudFlags   []Flag
	ExecProcess func(c *CliSC) error
	SubComs     []*CliSC
	CrudOps     []string
	CrudPK      []string
	InclAccID   bool
}

// Flag represents a CLI Flag
type Flag struct {
	Type        string
	ShortName   string
	Name        string
	Description string
}

// Add adds to the main CLI array
func (c *CliSC) Add() *CliSC {
	CliArr = append(CliArr, c)
	return c
}

// Make makes the subcommand properties
func (c *CliSC) Make() *CliSC {
	c.Sc = flaggy.NewSubcommand(c.Name)
	c.Sc.Description = c.Description

	c.Vals = map[string]interface{}{}
	for _, f := range c.Flags {
		switch f.Type {
		case "bool":
			val := false
			c.Sc.Bool(&val, f.ShortName, f.Name, f.Description)
			c.Vals[f.Name] = &val
		default:
			val := ""
			c.Sc.String(&val, f.ShortName, f.Name, f.Description)
			c.Vals[f.Name] = &val
		}
	}

	for i, f := range c.PosFlags {
		val := ""
		c.Sc.AddPositionalValue(&val, f.Name, i+1, true, f.Description)
		c.Vals[f.Name] = &val
	}

	listFlagsAllowed := map[string]string{
		"hostname":   "",
		"name":       "",
		"type":       "",
		"account_id": "",
		"email":      "",
		"role":       "",
		"job_name":   "",
		"active":     "",
	}

	for _, op := range c.CrudOps {
		switch op {
		case "add":
			addSC := flaggy.NewSubcommand("add")
			addSC.Description = F("add a new %s", c.Singular)
			for _, f := range c.CrudFlags {
				val := new(string)
				addSC.String(val, "", f.Name, f.Description)
				c.Vals["add="+f.Name] = val
			}
			c.Sc.AttachSubcommand(addSC, 1)
		case "update":
			addSC := flaggy.NewSubcommand("update")
			addSC.Description = F("update existing %s", c.Singular)
			id := ""
			addSC.String(&id, "", "id", F("update the %s with the provided id", c.Singular))
			c.Vals["update=id"] = &id
			if c.InclAccID {
				accID := ""
				addSC.String(&accID, "", "account_id", F("update the %s with the provided account id", c.Singular))
				c.Vals["update=account_id"] = &accID
			}
			for _, f := range c.CrudFlags {
				val := new(string)
				if _, ok := c.Vals["update="+f.Name]; ok {
					continue
				}
				addSC.String(val, "", f.Name, f.Description)
				c.Vals["update="+f.Name] = val
			}
			c.Sc.AttachSubcommand(addSC, 1)
		case "list":
			listSC := flaggy.NewSubcommand("list")
			listSC.Description = F("list %ss", c.Singular)

			limit := ""
			listSC.String(&limit, "l", "limit", "limit records. (max: 500)")
			c.Vals["list=limit"] = &limit

			last := ""
			listSC.String(&last, "", "last", "show the N most recent records. (max: 100)")
			c.Vals["list=last"] = &last

			id := ""
			listSC.String(&id, "", "id", F("show the %s with the provided id", c.Singular))
			c.Vals["list=id"] = &id

			for _, f := range c.CrudFlags {
				if _, ok := listFlagsAllowed[f.Name]; ok {
					val := ""
					listSC.String(&val, "", f.Name, F("filter results by %s", f.Name))
					c.Vals["list="+f.Name] = &val
				}
			}
			c.Sc.AttachSubcommand(listSC, 1)
		case "remove":
			id := ""
			removeSC := flaggy.NewSubcommand("remove")
			removeSC.Description = F("remove a %s", c.Singular)
			removeSC.String(&id, "", "id", F("removes the %s with the provided id", c.Singular))
			c.Vals["remove=id"] = &id
			if c.InclAccID {
				accID := ""
				removeSC.String(&accID, "", "account_id", F("removes the %s with the provided account id", c.Singular))
				c.Vals["remove=account_id"] = &accID
			}
			c.Sc.AttachSubcommand(removeSC, 1)
		default:
			continue
		}
	}

	for _, s := range c.SubComs {
		s.Make()
		c.Sc.AttachSubcommand(s.Sc, 1)
		if s.ExecProcess == nil {
			s.ExecProcess = c.ExecProcess
		}
	}

	return c
}

// CliProcess processes the cli objects
func CliProcess() (bool, error) {

	allBlanks := func(m map[string]interface{}) bool {
		if len(m) == 0 { // no flags
			return false
		}

		blankCnt := 0
		for k, v := range m {
			switch v.(type) {
			case *bool:
				b := *v.(*bool)
				if !b {
					blankCnt++
				}
				m[k] = b
			case *int:
				i := *v.(*int)
				if i == 0 {
					blankCnt++
				}
				m[k] = i
			default:
				s := *v.(*string)
				if s == "" {
					blankCnt++
				}
				m[k] = s
			}
		}
		return blankCnt == len(m)
	}

	for _, cObj := range CliArr {
		if cObj.Sc.Used {
			for _, sc2 := range cObj.Sc.Subcommands {
				if sc2.Used {
					for _, scCli := range cObj.SubComs {
						if scCli.Name == sc2.Name {
							for k, v := range scCli.Vals {
								cObj.Vals[k] = v
							}
						}
					}
					if sc2.Name == "list" && *cObj.Vals["list=limit"].(*string) == "" {
						defLimit := "20"
						cObj.Vals["list=limit"] = &defLimit
					}
				}
			}

			if allBlanks(cObj.Vals) {
				return false, nil
			}

			// delete blanks, prepare values
			for k, v := range cObj.Vals {
				val := cast.ToString(v)
				if val == "" {
					delete(cObj.Vals, k)
					continue
				}
				keyArr := strings.Split(k, "=")
				if len(keyArr) == 2 {
					cObj.Vals[keyArr[1]] = v
					delete(cObj.Vals, k)
					k = keyArr[1]
				}

				// try int
				valInt, err := cast.ToIntE(v)
				if err == nil {
					cObj.Vals[k] = valInt
				}
			}

			err := cObj.ExecProcess(cObj)
			if err != nil {
				err = Error(err)
			}
			return true, err
		}
	}

	return false, nil
}

// ListWhere get the list where fields/values
func (c *CliSC) ListWhere() map[string]interface{} {
	where := map[string]interface{}{}
	for _, flag := range c.CrudFlags {
		if v, ok := c.Vals[flag.Name]; ok {
			where[flag.Name] = v
		}
	}
	return where
}

// UsedSC returns the name of the used subcommand
func (c *CliSC) UsedSC() string {
	for _, sc2 := range c.Sc.Subcommands {
		if !sc2.Used {
			continue
		}
		return sc2.Name
	}
	return ""
}
