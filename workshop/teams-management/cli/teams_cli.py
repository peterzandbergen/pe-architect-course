#!/usr/bin/env python3
"""
Teams CLI - A simple command-line interface for the Teams API
"""

import argparse
import json
import sys
import requests
import os
import logging
from typing import Optional

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
)

API_BASE_URL = "http://teams-api.127.0.0.1.sslip.io"

class TeamsAPI:

    def __init__(self, base_url: str = API_BASE_URL, json_output: Optional[bool] = False, token: Optional[str] = None):
        self.base_url = base_url
        self.json_output = json_output
        self.token = token
        
    def _make_request(self, method: str, endpoint: str, data: Optional[dict] = None) -> dict:
        """Make HTTP request to the API"""
        url = f"{self.base_url}{endpoint}"

        session = requests.Session()
        if self.token:
            logging.info("Using authentication token for API requests")
            session.headers.update({
                "Authorization": f"Bearer {self.token}"
            })
        else:
            logging.info("No authentication token provided, making unauthenticated requests")

        try:
            if method == "GET":
                response = session.get(url)
            elif method == "POST":
                session.headers.update({"Content-Type": "application/json"})
                response = requests.post(url, json=data)
            elif method == "DELETE":
                response = requests.delete(url)
            else:
                raise ValueError(f"Unsupported method: {method}")
                
            response.raise_for_status()
            return response.json()
        except requests.exceptions.ConnectionError:
            print(f"❌ Error: Could not connect to API at {self.base_url}")
            print("   Make sure the Teams API is running")
            sys.exit(1)
        except requests.exceptions.HTTPError as e:
            if response.status_code == 400:
                error_detail = response.json().get("detail", "Bad request")
                print(f"❌ Error: {error_detail}")
            elif response.status_code == 404:
                print("❌ Error: Team not found")
            else:
                print(f"❌ HTTP Error {response.status_code}: {e}")
            sys.exit(1)
        except Exception as e:
            print(f"❌ Unexpected error: {e}")
            sys.exit(1)

    def health_check(self):
        """Check API health"""
        result = self._make_request("GET", "/health")

        if self.json_output:
            self.print_json(result)
            return
        
        status = result.get("status", "unknown")
        teams_count = result.get("teams_count", 0)
        print(f"✅ API Status: {status}")
        print(f"📊 Teams Count: {teams_count}")

    def create_team(self, name: str):
        """Create a new team"""
        result = self._make_request("POST", "/teams", {"name": name})

        if self.json_output:
            self.print_json(result)
            return
        
        print(f"✅ Created team: {result['name']}")
        print(f"🆔 Team ID: {result['id']}")
        print(f"📅 Created: {result['created_at']}")

    def list_teams(self):
        """List all teams"""
        teams = self._make_request("GET", "/teams")
        if not teams:
            print("📭 No teams found")
            return

        if self.json_output:
            self.print_json(teams)
            return
            
        print(f"📋 Found {len(teams)} team(s):")
        print("-" * 60)
        for team in teams:
            self.print_team(team)
            print("-" * 60)

    def get_team(self, team_id: str):
        """Get a specific team by ID"""
        team = self._make_request("GET", f"/teams/{team_id}")

        if self.json_output:
            self.print_json(team)
            return
        
        self.print_team(team)

    def delete_team(self, team_id: str):
        """Delete a team"""
        result = self._make_request("DELETE", f"/teams/{team_id}")

        if self.json_output:
            self.print_json(result)
            return

        print(f"✅ {result['message']}")

    def print_team(self, team):
        """Helper method to print team details"""
        print(f"🏷️  Name: {team['name']}")
        print(f"🆔 ID: {team['id']}")
        print(f"📅 Created: {team['created_at']}")

    def print_json(self, data):
        """Helper method to print JSON data in a readable format"""
        print(json.dumps(data, indent=2))

def main():
    parser = argparse.ArgumentParser(
        description="Teams CLI - Manage teams via the Teams API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  teams-cli health                    # Check API health
  teams-cli create "Backend Team"     # Create a new team
  teams-cli list                      # List all teams
  teams-cli get <team-id>            # Get specific team
  teams-cli delete <team-id>         # Delete a team
        """
    )
    
    parser.add_argument(
        "--url", 
        default=os.environ.get("TEAMS_API_URL", API_BASE_URL),
        help=f"API base URL (default: {API_BASE_URL})"
    )
    
    parser.add_argument(
        "--json", 
        default=os.environ.get("TEAMS_CLI_JSON", "false").lower() == "true",
        help=f"Output JSON",
        action='store_true'
    )

    parser.add_argument(
        "--token",
        default=os.environ.get("TEAMS_API_TOKEN"),
        help="API authentication token (optional)"
    )
    
    subparsers = parser.add_subparsers(dest="command", help="Available commands")
    
    # Health command
    subparsers.add_parser("health", help="Check API health")
    
    # Create command
    create_parser = subparsers.add_parser("create", help="Create a new team")
    create_parser.add_argument("name", help="Team name")
    
    # List command
    subparsers.add_parser("list", help="List all teams")
    
    # Get command
    get_parser = subparsers.add_parser("get", help="Get a specific team")
    get_parser.add_argument("team_id", help="Team ID")
    
    # Delete command
    delete_parser = subparsers.add_parser("delete", help="Delete a team")
    delete_parser.add_argument("team_id", help="Team ID")
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return
    
    # Initialize API client
    api = TeamsAPI(args.url, args.json, args.token)
    
    # Execute command
    try:
        if args.command == "health":
            api.health_check()
        elif args.command == "create":
            api.create_team(args.name)
        elif args.command == "list":
            api.list_teams()
        elif args.command == "get":
            api.get_team(args.team_id)
        elif args.command == "delete":
            api.delete_team(args.team_id)
    except KeyboardInterrupt:
        print("\n👋 Goodbye!")
        sys.exit(0)

if __name__ == "__main__":
    main()
