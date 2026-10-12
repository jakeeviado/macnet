import re
from typing import Dict, List


class UserProcessor:
    def __init__(self, min_age=18):
        self.min_age = min_age
        self._cache = {}

    def validate_email(self, email: str) -> bool:
        pattern = r"^[\w\.-]+@[\w\.-]+\.\w+$"
        return bool(re.match(pattern, email))

    def process_users(self, users: List[Dict]) -> List[Dict]:
        valid_users = []

        for u in users:
            if u.get("age", 0) < self.min_age:
                continue

            email = u.get("email", "")
            if not self.validate_email(email):
                print(f"Skipping {u.get('name')}: invalid email")
                continue

            # Normalize names
            u["name"] = u.get("name", "").strip().title()
            valid_users.append(u)

        return valid_users

    def get_stats(self, users: List[Dict]) -> Dict:
        if not users:
            return {"count": 0, "avg_age": 0}

        ages = [u.get("age", 0) for u in users]
        return {
            "count": len(users),
            "avg_age": sum(ages) / len(ages),
            "min_age": min(ages),
            "max_age": max(ages),
        }


if __name__ == "__main__":
    processor = UserProcessor(min_age=21)

    test_users = [
        {"name": "alice", "age": 25, "email": "pandesal@example.com"},
        {"name": "bob", "age": 17, "email": "kutsinta@example.com"},
        {"name": "charlie", "age": 30, "email": "invalid-email"},
    ]

    valid = processor.process_users(test_users)
    stats = processor.get_stats(valid)

    print(f"Processed {stats['count']} users, avg age: {stats['avg_age']:.1f}")
