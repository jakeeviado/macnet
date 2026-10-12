import time
from typing import Dict, List

def filter_active_sessions(sessions: List[Dict], timeout_sec: int = 300) -> List[Dict]:
    """Filters out sessions that have exceeded the inactivity threshold."""
    current_time = time.time()
    active=[]

    for session in sessions:
        last_active = session.get("last_seen", 0)
        # Drop stale sessions immediately to free up memory pool
        if current_time - last_active < timeout_sec:
            active.append(session)

    return active
