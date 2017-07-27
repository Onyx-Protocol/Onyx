package com.chain.analytics;

import com.chain.exception.BadURLException;
import com.chain.exception.ChainException;
import com.chain.exception.ConnectivityException;
import com.chain.exception.HTTPException;
import com.chain.http.Client;
import com.google.gson.*;
import com.mchange.v2.c3p0.ComboPooledDataSource;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;

import java.beans.PropertyVetoException;
import java.sql.SQLException;
import java.io.FileReader;
import java.io.FileNotFoundException;
import java.util.concurrent.TimeUnit;

/**
 * Application is the Main class for the Chain Analytics
 * importing service.
 */
public class Application {
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

  private static final Logger logger = LogManager.getLogger();

  public static void main(String args[]) {
    final String chainUrl = System.getenv(ENV_CHAIN_URL);
    final String chainToken = System.getenv(ENV_CHAIN_TOKEN);
    final String databaseUrl = System.getenv(ENV_DATABASE_URL);

    if (chainUrl == null || "".equals(chainUrl)) {
      logger.fatal("missing {} environment variable", ENV_CHAIN_URL);
      System.exit(1);
    }
    if (databaseUrl == null || "".equals(databaseUrl)) {
      logger.fatal("missing {} environment variable", ENV_DATABASE_URL);
      System.exit(1);
    }
    if (args.length < 1) {
      logger.fatal("Usage: java com.chain.analytics.Application [command]");
      System.exit(1);
    }

    final Target target = createTarget(databaseUrl);
    switch (args[0]) {
      case "migrate":
        if (args.length != 2) {
          logger.fatal("Usage: java com.chain.analytics.Application migrate config.json");
          System.exit(1);
        }
        try {
          final Config newConfig = Config.readFromJSON(new FileReader(args[1]));

          target.migrate(newConfig);
          logger.info("Successfully migrated database to new config.");
        } catch (Config.InvalidConfigException | JsonSyntaxException | JsonIOException ex) {
          logger.fatal("Unable to load JSON configuration.", ex);
          System.exit(1);
        } catch (FileNotFoundException ex) {
          logger.fatal("Unable to find configuration file: {}", args[1]);
          System.exit(1);
        } catch (SQLException ex) {
          logger.fatal("Unable to perform migration", ex);
          System.exit(1);
        }
        break;

      case "run":
        run(target, chainUrl, chainToken);
        break;

      default:
        logger.fatal("Unknown command: {}", args[0]);
        System.exit(1);
    }
  }

  public static Target createTarget(final String databaseUrl) {
    try {
      // Use a connection pool.
      ComboPooledDataSource ds = new ComboPooledDataSource();
      ds.setDriverClass("oracle.jdbc.driver.OracleDriver");
      ds.setJdbcUrl(databaseUrl);
      ds.setTestConnectionOnCheckout(true);
      return new Target(ds);
    } catch (PropertyVetoException ex) {
      logger.fatal("Unable to setup JDBC. Is the Oracle driver in the classpath?", ex);
      System.exit(1);
    } catch (Config.InvalidConfigException | SQLException ex) {
      logger.fatal("Unable to load stored configuration.", ex);
      System.exit(1);
    }
    return null;
  }

  public static void run(final Target target, final String chainUrl, final String chainToken) {
    //
    // Setup the importer. The majority of connectivity and
    // configuration errors should be caught here.
    //
    if (target.getConfig() == null) {
      logger.fatal("Missing Chain Analytics configuration. Have you configured it yet?");
      System.exit(1);
    }
    Importer importer = null;
    try {
      Client.Builder clientBuilder =
          new Client.Builder().addURL(chainUrl).setReadTimeout(120, TimeUnit.SECONDS);
      if (chainToken != null && chainToken.length() > 0) {
        clientBuilder.setAccessToken(chainToken);
      }
      final Client client = clientBuilder.build();

      importer = Importer.connect(client, target);
    } catch (BadURLException ex) {
      logger.fatal("Unable to parse the Chain Core URL provided \"{}\".", chainUrl, ex);
      System.exit(1);
    } catch (HTTPException | ConnectivityException ex) {
      logger.fatal(
          "Unable to connect to Chain Core at the configured URL \"{}\". "
              + "Double check that the URL is correct and reachable.",
          chainUrl,
          ex);
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
