import { useEffect, useRef, useState } from "react";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";

function App() {
  const terminalRef = useRef(null);
  const socketRef = useRef(null);
  const [isMobile, setIsMobile] = useState(window.innerWidth < 768);

  useEffect(() => {
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      theme: { background: "#000" },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    term.open(terminalRef.current);
    fitAddon.fit();

    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    const host = window.location.hostname;

    const socket = new WebSocket(`${protocol}://${host}:8080/terminal`);
    socket.binaryType = "arraybuffer";
    socketRef.current = socket;

    socket.onmessage = (event) => {
      term.write(new Uint8Array(event.data));
    };

    term.onData((data) => {
      if (socket.readyState === WebSocket.OPEN) {
        socket.send(data);
      }
    });

    socket.onopen = () => {
      const resize = () => {
        fitAddon.fit();

        if (socket.readyState === WebSocket.OPEN) {
          socket.send(
            JSON.stringify({
              type: "resize",
              cols: term.cols,
              rows: term.rows,
            })
          );
        }
      };

      window.addEventListener("resize", resize);
      resize();

      socket.onclose = () => {
        window.removeEventListener("resize", resize);
      };
    };

    const handleResize = () => {
      setIsMobile(window.innerWidth < 768);
    };

    window.addEventListener("resize", handleResize);

    return () => {
      socket.close();
      term.dispose();
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  // ✅ Now available globally in component
  const sendKey = (key) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(key);
    }
  };

  return (
    <div style={containerStyle}>
      <div ref={terminalRef} style={terminalStyle(isMobile)} />

      {isMobile && (
        <div style={controlsStyle}>
          <button style={btn} onClick={() => sendKey("\x1b[A")}>↑</button>
          <button style={btn} onClick={() => sendKey("\x1b[B")}>↓</button>
          <button style={btn} onClick={() => sendKey("\x1b[D")}>←</button>
          <button style={btn} onClick={() => sendKey("\x1b[C")}>→</button>
          <button style={btn} onClick={() => sendKey("\t")}>Tab</button>
          <button style={btn} onClick={() => sendKey("\r")}>Enter</button>
        </div>
      )}
    </div>
  );
}

const containerStyle = {
  height: "100vh",
  width: "100vw",
  display: "flex",
  flexDirection: "column",
  background: "black",
};

const terminalStyle = (isMobile) => ({
  flex: 1,
  width: "100%",
  overflow: "hidden",
});

const controlsStyle = {
  display: "flex",
  justifyContent: "space-around",
  alignItems: "center",
  padding: "10px",
  background: "#111",
};

const btn = {
  padding: "10px 14px",
  fontSize: "16px",
  background: "#222",
  color: "white",
  border: "1px solid #444",
  borderRadius: "6px",
};

export default App;