import base64
import unittest

from signal_collector.skillsmp import SkillsMPCollector


class FakeClient:
    def __init__(self) -> None:
        self.requested_url = ""
        self.headers = None

    def get_text(self, url: str, headers=None) -> str:
        raise AssertionError("GitHub skill evidence should use the JSON contents API")

    def get_json(self, url: str, headers=None) -> dict:
        if url.endswith("/repos/owner/repo"):
            return {"full_name": "owner/repo"}
        if "/contents/skills/mcp/SKILL.md?ref=main" in url:
            content = "# MCP Inspector\n\nInstall with uv and run against a local server."
            return {"encoding": "base64", "content": base64.b64encode(content.encode()).decode()}
        if "/repos/owner/repo/releases?" in url:
            return []
        self.requested_url = url
        self.headers = headers
        return {
            "success": True,
            "data": {
                "skills": [
                    {
                        "id": "owner-repo-skill",
                        "name": "MCP Inspector",
                        "description": "Inspect MCP servers.",
                        "githubUrl": "https://github.com/owner/repo/tree/main/skills/mcp",
                        "skillUrl": "https://skillsmp.com/creators/owner/repo/skills-mcp",
                        "stars": 42,
                        "updatedAt": "1779015027",
                    }
                ]
            },
        }


class SkillsMPCollectorTests(unittest.TestCase):
    def test_searches_catalog_and_fetches_linked_github_skill_evidence(self) -> None:
        client = FakeClient()
        collector = SkillsMPCollector(client, api_key="secret", github_token="github-token")

        candidates = collector.search("mcp", limit=5)

        self.assertIn("q=mcp", client.requested_url)
        self.assertEqual(client.headers, {"Authorization": "Bearer secret"})
        self.assertEqual(candidates[0].url, "https://github.com/owner/repo/tree/main/skills/mcp")
        detail = collector.fetch_detail(candidates[0])
        self.assertFalse(detail.requires_github_verification)
        self.assertEqual(detail.evidence_class, "original_documentation")
        self.assertIn("Install with uv", detail.excerpt)
        self.assertEqual(detail.source_url, candidates[0].url)
