package {{.PackageName}}

{{- range $EntityTypeName, $Callers := .Data }}
var _{{$EntityTypeName}}_callerMap = entity.Calls{
    Where: map[entity.Caller]entity.QueryFunc{
    {{- range $caller, $query := $Callers}}
        entity.Caller{File: "{{$caller.Filename}}", Line: {{$caller.Line}}}: func(helper entity.Helper, query *bun.SelectQuery, args ...any) {
            query.Where("{{$query.Query}}", {{join $query.Args ", "}})
        },
    {{end -}}
    },
}
{{ end -}}

func init() {
{{- range $EntityTypeName, $_ := .Data }}
    entity.SetGlobalEntity[{{$EntityTypeName}}](_{{$EntityTypeName}}_callerMap)
{{end -}}
}
