#!/usr/bin/env python3
import json
import sys

# Read SGF file
with open('test_game_76776999.sgf', 'r') as f:
    sgf_content = f.read()

# Create the MCP request
# Note: This is a direct HTTP request for testing, not using MCP protocol
request = {
    "jsonrpc": "2.0",
    "method": "findMistakes",
    "params": {
        "sgf": sgf_content,
        "blunderThreshold": 0.15,
        "mistakeThreshold": 0.05,
        "inaccuracyThreshold": 0.02
    },
    "id": 1
}

# For testing, we'll need to use the actual MCP protocol
# This is just to demonstrate the structure
print("SGF file contains:", sgf_content.count(';B['), "black moves and", sgf_content.count(';W['), "white moves")
print("Total moves indicated in SGF:", sgf_content.count(';B[') + sgf_content.count(';W['))
print("\nFirst few moves:")
import re
moves = re.findall(r';[BW]\[[a-z]*\]', sgf_content)[:10]
for i, move in enumerate(moves):
    print(f"Move {i+1}: {move}")