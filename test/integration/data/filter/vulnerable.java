import java.sql.*;

public class VulnerableClass {
    public void getUser(String id) {
        String query = "SELECT * FROM users WHERE id = '" + id + "'";
    }
}
