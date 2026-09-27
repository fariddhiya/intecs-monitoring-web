export function isTokenValid() {
    const token = localStorage.getItem("token");
    if (!token) return false;

    try {
      const decoded = JSON.parse(atob(token));
      if (decoded.exp <= Date.now()) {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        return false;
      }
      return true;
    } catch (e) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      return false;
    }
}
