import java.sql.*;

public class TestClass {
    public void queryUser(String id) {
        String query = "SELECT * FROM users WHERE id = '" + id + "'";
    }
}
