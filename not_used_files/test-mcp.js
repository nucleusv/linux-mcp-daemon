const EventSource = require('eventsource');
const fetch = require('node-fetch');

const token = "my-test-token-123";
const sseUrl = "http://localhost:9091/sse";
let messageUrl = "";

const es = new EventSource(sseUrl, { headers: { "Authorization": `Bearer ${token}` } });

es.onmessage = async (e) => {
    const msg = JSON.parse(e.data);
    if (msg.endpoint) {
        messageUrl = new URL(msg.endpoint, sseUrl).toString();
        console.log("Connected! Message URL:", messageUrl);
        
        // Send tool call
        const payload = {
            jsonrpc: "2.0",
            id: 1,
            method: "tools/call",
            params: {
                name: "list_directory",
                arguments: { path: "/var/log", privileged: true }
            }
        };
        
        console.log("Calling list_directory...");
        const res = await fetch(messageUrl, {
            method: "POST",
            headers: { "Content-Type": "application/json", "Authorization": `Bearer ${token}` },
            body: JSON.stringify(payload)
        });
        if (!res.ok) console.error("POST failed", await res.text());
    } else {
        console.log("Received response:", JSON.stringify(msg, null, 2));
        if (msg.id === 1) {
            process.exit(0);
        }
    }
};

es.onerror = (e) => {
    console.error("SSE Error");
    process.exit(1);
};
