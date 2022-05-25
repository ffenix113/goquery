package {{.PackageName}}

{{- range $EntityTypeName, $Callers := .Data }}
var _{{$EntityTypeName}}_callerMap = goquery.Calls{
    Where: map[goquery.Caller]goquery.QueryFunc{
    {{- range $caller, $query := $Callers}}
        goquery.Caller{File: "{{$caller.Filename}}", Line: {{$caller.Line}}}: func(helper goquery.Helper, query *bun.SelectQuery, args ...any) {
            query.Where("{{$query.Query}}", {{join $query.Args ", "}})
        },
    {{end -}}
    },
}
{{ end -}}

func init() {
{{- range $EntityTypeName, $_ := .Data }}
    goquery.SetGlobalEntity[{{$EntityTypeName}}](_{{$EntityTypeName}}_callerMap)
{{end -}}
}
