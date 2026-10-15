#!/usr/bin/env python3
"""Check actual disposable PostgreSQL/HTTP responses against OpenAPI 3.1."""
import json
from pathlib import Path
import yaml
from jsonschema import Draft202012Validator, FormatChecker
root=Path(__file__).resolve().parents[1]
contract=yaml.safe_load((root/'content/projects/p2/openapi.yaml').read_text())
fixtures=root/'.data/p2-http-fixtures'
for name,schema in {'user.json':{'$ref':'#/components/schemas/User'},'tag.json':{'$ref':'#/components/schemas/Tag'},'link.json':{'$ref':'#/components/schemas/Link'},'untagged-link.json':{'$ref':'#/components/schemas/Link'},'links.json':{'type':'array','items':{'$ref':'#/components/schemas/Link'}},'not-found.json':{'type':'object','required':['error'],'properties':{'error':{'const':'not_found'}}}}.items():
    document={'$schema':'https://json-schema.org/draft/2020-12/schema','components':contract['components'],**schema}
    Draft202012Validator.check_schema(document)
    Draft202012Validator(document,format_checker=FormatChecker()).validate(json.loads((fixtures/name).read_text()))
    print('PASS actual HTTP response:',name)
print('PASS: six actual response fixtures match OpenAPI 3.1 schemas including UUID/URI/time formats')
