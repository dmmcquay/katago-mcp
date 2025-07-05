#!/usr/bin/env python3
import json
import subprocess
import sys

def send_mcp_request(process, method, params=None):
    """Send a JSON-RPC request to the MCP server"""
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params or {}
    }
    
    request_json = json.dumps(request)
    print(f"Sending: {request_json}", file=sys.stderr)
    
    process.stdin.write(request_json + '\n')
    process.stdin.flush()
    
    # Read response
    response_line = process.stdout.readline()
    print(f"Received: {response_line}", file=sys.stderr)
    
    return json.loads(response_line)

def main():
    # Read the SGF file
    with open('test_game_76776999.sgf', 'r') as f:
        sgf_content = f.read()
    
    print(f"SGF file length: {len(sgf_content)} characters", file=sys.stderr)
    
    # Start the MCP server
    env = {
        **subprocess.os.environ,
        'KATAGO_MCP_CONFIG': 'config.local.json'
    }
    
    process = subprocess.Popen(
        ['./katago-mcp'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
        env=env
    )
    
    try:
        # Initialize the connection
        init_response = send_mcp_request(process, "initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {
                "name": "test-client",
                "version": "1.0"
            }
        })
        print(f"Initialize response: {init_response}", file=sys.stderr)
        
        # Call findMistakes
        print("\nCalling findMistakes...", file=sys.stderr)
        find_mistakes_response = send_mcp_request(process, "tools/call", {
            "name": "findMistakes",
            "arguments": {
                "sgf": sgf_content,
                "maxVisits": 100
            }
        })
        
        # Extract the result
        if 'result' in find_mistakes_response:
            result = find_mistakes_response['result']
            if 'content' in result and len(result['content']) > 0:
                text_content = result['content'][0].get('text', '')
                print("\n=== ANALYSIS RESULT ===")
                print(text_content)
                
                # Check if it shows the correct number of moves
                if "Total moves: 271" in text_content:
                    print("\n✓ SUCCESS: Correctly analyzed all 271 moves!")
                elif "Total moves: 1" in text_content:
                    print("\n✗ FAILURE: Bug still present - only analyzed 1 move")
                else:
                    # Extract total moves from the output
                    import re
                    match = re.search(r'Total moves: (\d+)', text_content)
                    if match:
                        total_moves = int(match.group(1))
                        print(f"\n⚠ Found {total_moves} moves (expected 271)")
            else:
                print("No content in result")
        else:
            print(f"Error response: {find_mistakes_response}")
            
    finally:
        # Cleanup
        process.terminate()
        process.wait()

if __name__ == "__main__":
    main()