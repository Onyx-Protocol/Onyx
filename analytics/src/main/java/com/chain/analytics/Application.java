package analytics;

import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.http.Client;
import com.mchange.v2.c3p0.ComboPooledDataSource;

import java.beans.PropertyVetoException;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.SQLException;
import java.util.concurrent.TimeUnit;

import org.apache.logging.log4j.Logger;
import org.apache.logging.log4j.LogManager;

/**
 * Application is the Main class for the Chain Analytics
 * importing service.
 */
public class Application {
    // TODO(jackson): Allow configuration of custom columns.

    // Environment variable key to find the Chain Core's URL.
    public static final String ENV_CHAIN_URL = "CHAIN_URL";

    // Environment variable key to find the Chain Core access token
    // to use, if any.
    public static final String ENV_CHAIN_TOKEN = "CHAIN_ACCESS_TOKEN";

    // Environment variable key to find the JDBC URL to use to connect
    // to the Oracle database.
    // Example: jdbc:oracle:thin:username/password@127.0.0.1:1521/orcl
    public static final String ENV_DATABASE_URL = "DATABASE_URL";

    // The alias of the Chain Core transaction feed that the
    // importer uses.
    public static final String DEFAULT_FEED_ALIAS = "chain-analytics-importer";

    private static Logger logger = LogManager.getLogger();

    public static void main(String args[]) {
        String chainUrl = System.getenv(ENV_CHAIN_URL);
        String chainToken = System.getenv(ENV_CHAIN_TOKEN);
        String databaseUrl = System.getenv(ENV_DATABASE_URL);

        if (chainUrl == null || "".equals(chainUrl)) {
            logger.fatal("missing {} environment variable", ENV_CHAIN_URL);
            System.exit(1);
        }
        if (databaseUrl == null || "".equals(databaseUrl)) {
            logger.fatal("missing {} environment variable", ENV_DATABASE_URL);
            System.exit(1);
        }

        //
        // Setup the importer. The majority of connectivity and
        // configuration errors should be caught here.
        //
        Importer importer = null;
        try {
            Client.Builder clientBuilder = new Client.Builder()
                .addURL(chainUrl)
                .setReadTimeout(60, TimeUnit.SECONDS);
            if (chainToken != null && chainToken.length() > 0) {
                clientBuilder.setAccessToken(chainToken);
            }
            final Client client = clientBuilder.build();

            // Use a connection pool.
            ComboPooledDataSource ds = new ComboPooledDataSource();
            ds.setDriverClass("oracle.jdbc.driver.OracleDriver");
            ds.setJdbcUrl(databaseUrl);
            ds.setTestConnectionOnCheckout(true);

            importer = Importer.connect(client, ds, DEFAULT_FEED_ALIAS);
        } catch(BadURLException ex) {
            logger.fatal(
                    "Unable to parse the Chain Core URL provided \"{}\".",
                    chainUrl, ex);
            System.exit(1);
        } catch (PropertyVetoException ex) {
            logger.fatal("Unable to setup JDBC. Is the Oracle driver in the classpath?", ex);
            System.exit(1);
        } catch (HTTPException | ConnectivityException ex) {
            logger.fatal(
                    "Unable to connect to Chain Core at the configured URL \"{}\". " +
                    "Double check that the URL is correct and reachable.",
                    chainUrl, ex);
            System.exit(1);
        } catch (ChainException | SQLException ex) {
            logger.fatal("Unable to initialize importer.", ex);
            System.exit(1);
        }

        // If we make it this far, the importer is properly configured.
        // We can begin syncing. From now on, errors should not crash
        // the process unless they're persistent.
        logger.info("Transaction importer initialized");
        importer.process();
    }
}
