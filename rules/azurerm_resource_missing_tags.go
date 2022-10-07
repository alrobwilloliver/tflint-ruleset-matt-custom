// Based on AWS missing resource tag rule: https://github.com/terraform-linters/tflint-ruleset-aws/blob/master/docs/rules/aws_resource_missing_tags.md

package rules

import (
	"fmt"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/exp/maps"
)

// AzurermResourceMissingTagsRule checks whether resources are tagged correctly
type AzurermResourceMissingTagsRule struct {
	tflint.DefaultRule
}

type azurermResourceTagsRuleConfig struct {
	Tags    []string `hclext:"tags"`
	Exclude []string `hclext:"exclude,optional"`
}

const (
	tagsAttributeName = "tags"
)

// NewAzurermResourceMissingTagsRule returns new rules for all resources that support tags
func NewAzurermResourceMissingTagsRule() *AzurermResourceMissingTagsRule {
	return &AzurermResourceMissingTagsRule{}
}

// Name returns the rule name
func (r *AzurermResourceMissingTagsRule) Name() string {
	return "azurerm_resource_missing_tags"
}

// Enabled returns whether the rule is enabled by default
func (r *AzurermResourceMissingTagsRule) Enabled() bool {
	return false
}

// Severity returns the rule severity
func (r *AzurermResourceMissingTagsRule) Severity() tflint.Severity {
	return tflint.NOTICE
}

// Link returns the rule reference link
func (r *AzurermResourceMissingTagsRule) Link() string {
	//return project.ReferenceLink(r.Name())
	return ""
}

// Check checks resources for missing tags
func (r *AzurermResourceMissingTagsRule) Check(runner tflint.Runner) error {
	config := azurermResourceTagsRuleConfig{}
	if err := runner.DecodeRuleConfig(r.Name(), &config); err != nil {
		return err
	}

	for _, resourceType := range Resources {
		// Skip this resource if its type is excluded in configuration
		if stringInSlice(resourceType, config.Exclude) {
			continue
		}

		resources, err := runner.GetResourceContent(resourceType, &hclext.BodySchema{
			Attributes: []hclext.AttributeSchema{{Name: tagsAttributeName}},
		}, nil)
		if err != nil {
			return err
		}

		for _, resource := range resources.Blocks {
			if attribute, ok := resource.Body.Attributes[tagsAttributeName]; ok {
				value, _ := attribute.Expr.Value(&hcl.EvalContext{})

				wantType := cty.DynamicPseudoType

				runner.EvaluateExpr(attribute.Expr, &value, &tflint.EvaluateExprOption{WantType: &wantType})
				err = runner.EnsureNoError(err, func() error {
					r.emitIssue(runner, value, config, attribute.Expr.Range())
					return nil
				})
				if err != nil {
					return err
				}
			} else {
				logger.Debug("Walk `%s` resource", resource.Labels[0]+"."+resource.Labels[1])
				r.emitIssue(runner, cty.NilVal, config, resource.DefRange)
			}
		}
	}
	return nil
}

func (r *AzurermResourceMissingTagsRule) emitIssue(runner tflint.Runner, tags cty.Value, config azurermResourceTagsRuleConfig, location hcl.Range) {
	if tags.IsNull() {
		wantedString := strings.Join(config.Tags, ", ")
		issue := fmt.Sprintf("The resource is missing the following tags: %s.", wantedString)
		runner.EmitIssue(r, issue, location)
		return
	}

	mapValue := tags.AsValueMap()
	emptyMissing := make(map[string]struct{})
	tagsAlreadyIncluded := make(map[string]struct{})

	missing := evaluateMissingTags(mapValue, config, emptyMissing, tagsAlreadyIncluded)

	if len(missing) > 0 {
		wanted := make([]string, 0, len(missing))
		for tag := range missing {
			wanted = append(wanted, tag)
		}
		wantedString := strings.Join(wanted, ", ")
		issue := fmt.Sprintf("The resource is missing the following tags: %s.", wantedString)
		runner.EmitIssue(r, issue, location)
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func evaluateMissingTags(mapValue map[string]cty.Value, config azurermResourceTagsRuleConfig, missing map[string]struct{}, tagsAlreadyIncluded map[string]struct{}) map[string]struct{} {
	for _, requiredTag := range config.Tags {
		for tagName, attributeValue := range mapValue {
			if attributeValue.Type().IsObjectType() {
				// recursively go one level deeper as this is a nested tag
				var nestedMissingTags map[string]struct{}
				nestedMissingTags = evaluateMissingTags(attributeValue.AsValueMap(), config, missing, tagsAlreadyIncluded)
				maps.Copy(nestedMissingTags, missing)
			} else if attributeValue.Type().IsPrimitiveType() {
				// the value is a string, number or bool so don't go one level deeper
				// evaluate if the tag matches any of the required tags
				if tagName == requiredTag {
					delete(missing, tagName)
					tagsAlreadyIncluded[tagName] = struct{}{}
					// the tag name is included in the required strings, skip to the next required tag
					continue
				}

				// check that the tag name is not included in all the tags on the map
				_, tagNameIncludedAtThisLayer := mapValue[requiredTag]
				// check that the tag name is not included at a different layer
				_, tagNameIncludedAtDifferentLayer := tagsAlreadyIncluded[requiredTag]
				if !tagNameIncludedAtThisLayer && !tagNameIncludedAtDifferentLayer {
					// append the tag as missing
					missing[requiredTag] = struct{}{}
					// the required tag is missing so skip to the next required tag
				}
			}
		}
	}
	return missing
}
