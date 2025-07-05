#!/usr/bin/env python3
import subprocess
import json
import time

# Read the SGF file
with open('test_game_76776999.sgf', 'r') as f:
    sgf_content = f.read()

print(f"Testing with SGF file: test_game_76776999.sgf")
print(f"SGF length: {len(sgf_content)} characters")

# Count moves in SGF
move_count = sgf_content.count(';B[') + sgf_content.count(';W[')
print(f"Approximate move count from SGF: {move_count}")

# Create a simple test using the mcp client
mcp_command = [
    'mcp', 'call-tool', 
    '--server', 'npx', '-y', 'katago-mcp', '--',
    '--tool', 'findMistakes',
    '--arguments', json.dumps({
        "sgf": sgf_content,
        "maxVisits": 50
    })
]

print("\nRunning findMistakes analysis...")
print("Command:", ' '.join(mcp_command[:6]) + '...')

start_time = time.time()

try:
    result = subprocess.run(
        mcp_command,
        capture_output=True,
        text=True,
        timeout=300  # 5 minute timeout
    )
    
    elapsed = time.time() - start_time
    print(f"\nAnalysis completed in {elapsed:.1f} seconds")
    
    if result.returncode == 0:
        output = result.stdout
        
        # Check for key indicators
        if "Total moves: 271" in output:
            print("\n✅ SUCCESS: Analysis correctly shows 271 total moves!")
        elif "Total moves: 1" in output:
            print("\n❌ FAILURE: Bug still present - only analyzed 1 move")
        else:
            # Try to extract the total moves
            import re
            match = re.search(r'Total moves: (\d+)', output)
            if match:
                total = int(match.group(1))
                if total == move_count or abs(total - move_count) < 5:
                    print(f"\n✅ SUCCESS: Analysis shows {total} total moves (expected ~{move_count})")
                else:
                    print(f"\n⚠️  WARNING: Analysis shows {total} total moves (expected ~{move_count})")
            else:
                print("\n❓ Could not find 'Total moves:' in output")
        
        # Show a preview of the output
        print("\n--- Output Preview (first 1000 chars) ---")
        print(output[:1000])
        if len(output) > 1000:
            print(f"... (truncated, total length: {len(output)} chars)")
            
    else:
        print(f"\nError running analysis: {result.returncode}")
        print("STDERR:", result.stderr)
        
except subprocess.TimeoutExpired:
    print("\n❌ Analysis timed out after 5 minutes")
except Exception as e:
    print(f"\n❌ Error: {e}")