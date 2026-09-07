package engine

import (
	"github.com/ju4n97/hclapi/internal/config"
	"github.com/ju4n97/hclapi/internal/problem"
	"github.com/ju4n97/hclapi/internal/runtime"
	"github.com/ju4n97/hclapi/internal/validator"
)

// validateRequest enforces ingress constraints across path, query, headers, and body fields.
func validateRequest(execCtx *runtime.ExecutionContext, rules config.RequestRules) []problem.InvalidParam {
	var invalidParams []problem.InvalidParam

	if len(rules.PathFields) > 0 {
		invalidParams = append(invalidParams, validator.ValidateStringMap(execCtx.Request.Path, rules.PathFields)...)
	}

	if len(rules.QueryFields) > 0 {
		invalidParams = append(invalidParams, validator.ValidateStringMap(execCtx.Request.Query, rules.QueryFields)...)
	}

	if len(rules.HeaderFields) > 0 {
		invalidParams = append(invalidParams, validator.ValidateHeaders(execCtx.Request.Headers, rules.HeaderFields)...)
	}

	if len(rules.BodyFields) > 0 {
		bodyMap, ok := execCtx.Request.Body.(map[string]any)
		if !ok {
			if execCtx.Request.Body == nil {
				bodyMap = make(map[string]any)
			} else {
				return append(invalidParams, problem.InvalidParam{
					Name:   "body",
					Reason: "request body must be a JSON object",
				})
			}
		}

		normalizedBody, errs := validator.ValidateBody(bodyMap, rules.BodyFields)
		if len(errs) > 0 {
			invalidParams = append(invalidParams, errs...)
		} else {
			execCtx.Request.Body = normalizedBody
		}
	}

	return invalidParams
}
