package rules

import (
	"testing"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_AzurermResourceMissingTags(t *testing.T) {
	cases := []struct {
		Name     string
		Content  string
		Config   string
		Expected helper.Issues
	}{
		{
			Name: "Wanted tags: Bar,Foo, found: bar,foo",
			Content: `
		resource "azurerm_resource_group" "az_rg_1" {
		  name = "test_rg"
		  location = "West Europe"
		}`,
			Config: `
		rule "azurerm_resource_missing_tags" {
		  enabled = true
		  tags = ["Foo", "Bar"]
		}`,
			Expected: helper.Issues{
				{
					Rule:    NewAzurermResourceMissingTagsRule(),
					Message: "The resource is missing the following tags: Foo, Bar.",
					Range: hcl.Range{
						Filename: "module.tf",
						Start:    hcl.Pos{Line: 2, Column: 3},
						End:      hcl.Pos{Line: 2, Column: 46},
					},
				},
			},
		},
		{
			Name: "Wanted tags: Bar,Foo, found: bar,foo",
			Content: `
		resource "azurerm_resource_group" "az_rg_1" {
		  name = "test_rg"
		  location = "West Europe"
		  tags = {
		    foo = "bar"
		    bar = "baz"
		  }
		}`,
			Config: `
		rule "azurerm_resource_missing_tags" {
		  enabled = true
		  tags = ["Foo", "Bar"]
		}`,
			Expected: helper.Issues{
				{
					Rule:    NewAzurermResourceMissingTagsRule(),
					Message: "The resource is missing the following tags: Foo, Bar.",
					Range: hcl.Range{
						Filename: "module.tf",
						Start:    hcl.Pos{Line: 5, Column: 12},
						End:      hcl.Pos{Line: 8, Column: 6},
					},
				},
			},
		},
		{
			Name: "No tags",
			Content: `
		resource "azurerm_resource_group" "az_rg_1" {
		  name = "test_rg"
		  location = "West Europe"
		}`,
			Config: `
		rule "azurerm_resource_missing_tags" {
		  enabled = true
		  tags = ["Foo", "Bar"]
		}`,
			Expected: helper.Issues{
				{
					Rule:    NewAzurermResourceMissingTagsRule(),
					Message: "The resource is missing the following tags: Foo, Bar.",
					Range: hcl.Range{
						Filename: "module.tf",
						Start:    hcl.Pos{Line: 2, Column: 3},
						End:      hcl.Pos{Line: 2, Column: 46},
					},
				},
			},
		},
		{
			Name: "Tags are correct",
			Content: `
				resource "azurerm_resource_group" "az_rg_1" {
				  name = "test_rg"
				  location = "West Europe"
				  tags = {
					Bar = "ba"
					Foo = "waa"
				  }
				}`,
			Config: `
				rule "azurerm_resource_missing_tags" {
				  enabled = true
				  tags = ["Foo", "Bar"]
				}`,
			Expected: helper.Issues{},
		},
		{
			Name: "Should detect missing tags in nested common tags resource",
			Content: `
				resource "azurerm_resource_group" "rgp" {
					name     = "local.resource_group_name"
					location = "local.location"
					tags 	 = {
						common_tags = {
							"Application" = "PAL",
							"CostCenter"  = "PAL",
							"Environment" = "NonProd",
							"ManagedBy"   = "ws-nonprod-pal"
						}
					}
				}`,
			Config: `
				rule "azurerm_resource_missing_tags" {
					enabled = true
					tags = [ "Application", "CostCenter", "Environment", "ManagedBy" ]
				}`,
			Expected: helper.Issues{},
		},
		{
			Name: "Should detect no missing tags in nested common tags resource 3 layers deep",
			Content: `
					resource "azurerm_resource_group" "rgp" {
						name     = "local.resource_group_name"
						location = "local.location"
						tags 	 = {
							common_tags = {
								more_tags = {
									"Application" = "PAL",
									"CostCenter"  = "PAL",
									"Environment" = "NonProd",
									"ManagedBy"   = "ws-nonprod-pal"
								}
							}
						}
					}`,
			Config: `
					rule "azurerm_resource_missing_tags" {
						enabled = true
						tags = [ "Application", "CostCenter", "Environment", "ManagedBy" ]
					}`,
			Expected: helper.Issues{},
		},
		{
			Name: "Should detect missing tags in nested common tags resource 3 layers deep",
			Content: `
					resource "azurerm_resource_group" "rgp" {
						name     = "local.resource_group_name"
						location = "local.location"
						tags 	 = {
							common_tags = {
								more_tags = {
									"Application" = "PAL",
									"CostCenter"  = "PAL",
									"Environment" = "NonProd"
								}
							}
						}
					}`,
			Config: `
					rule "azurerm_resource_missing_tags" {
						enabled = true
						tags = [ "Application", "CostCenter", "Environment", "ManagedBy" ]
					}`,
			Expected: helper.Issues{
				{
					Rule:    &AzurermResourceMissingTagsRule{},
					Message: "The resource is missing the following tags: ManagedBy.",
					Range: hcl.Range{
						Filename: "module.tf",
						Start:    hcl.Pos{Line: 5, Column: 16},
						End:      hcl.Pos{Line: 13, Column: 8},
					},
				},
			},
		},
		{
			Name: "Should detect no missing tags in nested common tags resource 3 layers deep at 2 different layers of abstraction",
			Content: `
			resource "azurerm_resource_group" "rgp" {
				name     = "local.resource_group_name"
				location = "local.location"
				tags 	 = {
					common_tags = {
						more_tags = {
							"Application" = "PAL",
							"CostCenter"  = "PAL",
						}
						"Environment" = "NonProd",
						"ManagedBy"   = "ws-nonprod-pal"
					}
				}
			}`,
			Config: `
			rule "azurerm_resource_missing_tags" {
				enabled = true
				tags = [ "Application", "CostCenter", "Environment", "ManagedBy" ]
			}`,
			Expected: helper.Issues{},
		},
		{
			Name: "Should detect no missing tags in nested common tags resource 3 layers deep at 3 different layers of abstraction",
			Content: `
			resource "azurerm_resource_group" "rgp" {
				name     = "local.resource_group_name"
				location = "local.location"
				tags 	 = {
					common_tags = {
						"Application" = "PAL",
						more_tags = {
							"CostCenter"  = "PAL",
						}
						"Environment" = "NonProd",
						"ManagedBy"   = "ws-nonprod-pal"
					}
				}
			}`,
			Config: `
			rule "azurerm_resource_missing_tags" {
				enabled = true
				tags = [ "Application", "CostCenter", "Environment", "ManagedBy" ]
			}`,
			Expected: helper.Issues{},
		},
		{
			Name: "Should detect missing tags in nested common tags resource 3 layers deep at different layers of abstraction",
			Content: `
			resource "azurerm_resource_group" "rgp" {
				name     = "local.resource_group_name"
				location = "local.location"
				tags 	 = {
					common_tags = {
						more_tags = {
							"Application" = "PAL",
						}
						"Environment" = "NonProd",
						"ManagedBy"   = "ws-nonprod-pal"
					}
				}
			}`,
			Config: `
			rule "azurerm_resource_missing_tags" {
				enabled = true
				tags = [ "Application", "CostCenter", "Environment", "ManagedBy" ]
			}`,
			Expected: helper.Issues{
				{
					Rule:    &AzurermResourceMissingTagsRule{},
					Message: "The resource is missing the following tags: CostCenter.",
					Range: hcl.Range{
						Filename: "module.tf",
						Start:    hcl.Pos{Line: 5, Column: 14},
						End:      hcl.Pos{Line: 13, Column: 6},
					},
				},
			},
		},
	}

	rule := NewAzurermResourceMissingTagsRule()

	for _, tc := range cases {
		runner := helper.TestRunner(t, map[string]string{"module.tf": tc.Content, ".tflint.hcl": tc.Config})

		if err := rule.Check(runner); err != nil {
			t.Fatalf("Unexpected error occurred: %s", err)
		}

		helper.AssertIssues(t, tc.Expected, runner.Issues)
	}
}
