import sys

with open("cmd/mcpd/rpc.go", "r") as f:
    lines = f.readlines()

def get_lines(start, end):
    return "".join(lines[start-1:end])

header = get_lines(1, 22)

switch_stmt = """
	switch req.Method {
	case "initialize":
		handleInitialize(&resp)
	case "notifications/initialized":
		return
	case "ping":
		// Ping is just an empty map, handled implicitly
	case "resources/list":
		handleResourcesList(&resp)
	case "resources/templates/list":
		handleResourcesTemplatesList(&resp)
	case "resources/read":
		handleResourcesRead(session, req, &resp)
	case "tools/list":
		handleToolsList(&resp)
	case "tools/call":
		handleToolsCall(session, req, &resp)
	default:
		resp.Error = map[string]interface{}{"code": -32601, "message": "Method not found"}
	}
"""

tail = get_lines(696, 700)

funcs = []

funcs.append("func handleInitialize(resp *JSONRPCResponse) {\n" + get_lines(24, 34) + "}\n")
funcs.append("func handleResourcesList(resp *JSONRPCResponse) {\n" + get_lines(41, 99) + "}\n")
funcs.append("func handleResourcesTemplatesList(resp *JSONRPCResponse) {\n" + get_lines(101, 116) + "}\n")
funcs.append("func handleResourcesRead(session *Session, req JSONRPCRequest, resp *JSONRPCResponse) {\n" + get_lines(118, 285) + "}\n")
funcs.append("func handleToolsList(resp *JSONRPCResponse) {\n" + get_lines(287, 573) + "}\n")
funcs.append("func handleToolsCall(session *Session, req JSONRPCRequest, resp *JSONRPCResponse) {\n" + get_lines(575, 691) + "}\n")

with open("cmd/mcpd/rpc.go", "w") as f:
    f.write(header + switch_stmt + tail + "\n" + "\n".join(funcs))

