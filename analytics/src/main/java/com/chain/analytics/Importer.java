package analytics;

import com.chain.http.Client;
import com.chain.api.Transaction;
import com.chain.exception.APIException;
import com.chain.exception.ChainException;
import com.google.gson.Gson;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.SQLException;
import java.sql.SQLSyntaxErrorException;
import java.sql.Timestamp;
import java.util.Arrays;
import java.util.Map;
import javax.sql.DataSource;

import org.apache.logging.log4j.Logger;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.ThreadContext;

/**
 * Importer is responsible for reading transactions from a Chain Core
 * transaction feed and writing them to an Oracle database.
 */
public class Importer {

    private static final String TRUE = "1";
    private static final String FALSE = "0";

    private static final Logger logger = LogManager.getLogger();
    private static final Gson gson = new Gson();

    private Client mChain;
    private DataSource mDataSource;
    private Transaction.Feed mFeed;
    private Schema mTransactionsTbl;
    private Schema mTransactionInputsTbl;
    private Schema mTransactionOutputsTbl;

    /**
     * connect initializes an Importer using the transaction feed specified
     * by the alias. If the feed doesn't yet exist, it will be created. It
     * does not begin syncing yet.
     * @param client    a client for the Chain Core
     * @param ds        the database to populate
     * @param feedAlias the alias of the transaction feed to use
     * @return          the initialized transaction importer
     */
    public static Importer connect(
            final Client client,
            final DataSource ds,
            final String feedAlias) throws ChainException, SQLException {
        // Create or load the transaction feed for the provided alias.
        try {
            Transaction.Feed.create(client, feedAlias, "");
        } catch (APIException ex) {
            // CH050 means the transaction feed already existed. If that's
            // the case, ignore the exception because we'll retrieve the
            // feed down below by its alias.
            if (!"CH050".equals(ex.code)) {
                logger.catching(ex);
                throw ex;
            }
            logger.info("Transaction feed {} already exists", feedAlias);
        }
        final Transaction.Feed feed = Transaction.Feed.getByAlias(client, feedAlias);
        logger.info("Using transaction feed {} starting at cursor {}", feed.id, feed.after);

        // Initialize the schema based on the configuration.
        final Importer importer = new Importer(client, ds, feed);
        importer.initializeSchema();
        return importer;
    }

    private Importer(final Client client, final DataSource ds, final Transaction.Feed feed) {
        mChain = client;
        mDataSource = ds;
        mFeed = feed;
    }

    void initializeSchema() throws SQLException {

        mTransactionsTbl = new Schema.Builder("transactions")
            .setPrimaryKey(Arrays.asList("id"))
            .addColumn("id", new Schema.Varchar2(64))
            .addColumn("block_height", new Schema.Integer())
            .addColumn("timestamp", new Schema.Timestamp())
            .addColumn("position", new Schema.Integer())
            .addColumn("local", new Schema.Boolean())
            .addColumn("reference_data", new Schema.Blob())
            .addColumn("data", new Schema.Blob())
            .build();

        mTransactionInputsTbl = new Schema.Builder("transaction_inputs")
            .setPrimaryKey(Arrays.asList("transaction_id", "index"))
            .addColumn("transaction_id", new Schema.Varchar2(64))
            .addColumn("index", new Schema.Integer())
            .addColumn("type", new Schema.Varchar2(64))
            .addColumn("asset_id", new Schema.Varchar2(64))
            .addColumn("asset_alias", new Schema.Varchar2(2000))
            .addColumn("asset_definition", new Schema.Blob())
            .addColumn("asset_tags", new Schema.Blob())
            .addColumn("local_asset", new Schema.Boolean())
            .addColumn("amount", new Schema.Integer())
            .addColumn("account_id", new Schema.Varchar2(64))
            .addColumn("account_alias", new Schema.Varchar2(2000))
            .addColumn("account_tags", new Schema.Blob())
            .addColumn("issuance_program", new Schema.Clob())
            .addColumn("reference_data", new Schema.Blob())
            .addColumn("local", new Schema.Boolean())
            .addColumn("spent_output_id", new Schema.Varchar2(64))
            .build();

        mTransactionOutputsTbl = new Schema.Builder("transaction_outputs")
            .setPrimaryKey(Arrays.asList("output_id"))
            .addColumn("transaction_id", new Schema.Varchar2(64))
            .addColumn("index", new Schema.Integer())
            .addColumn("output_id", new Schema.Varchar2(64))
            .addColumn("type", new Schema.Varchar2(64))
            .addColumn("purpose", new Schema.Varchar2(64))
            .addColumn("asset_id", new Schema.Varchar2(64))
            .addColumn("asset_alias", new Schema.Varchar2(2000))
            .addColumn("asset_definition", new Schema.Blob())
            .addColumn("asset_tags", new Schema.Blob())
            .addColumn("local_asset", new Schema.Boolean())
            .addColumn("amount", new Schema.Integer())
            .addColumn("account_id", new Schema.Varchar2(64))
            .addColumn("account_alias", new Schema.Varchar2(2000))
            .addColumn("account_tags", new Schema.Blob())
            .addColumn("control_program", new Schema.Clob())
            .addColumn("reference_data", new Schema.Blob())
            .addColumn("local", new Schema.Boolean())
            .addColumn("spent", new Schema.Boolean())
            .build();

        try (Connection conn = mDataSource.getConnection()) {
            createTableIfNotExists(conn, mTransactionsTbl.getDDLStatement());
            createTableIfNotExists(conn, mTransactionInputsTbl.getDDLStatement());
            createTableIfNotExists(conn, mTransactionOutputsTbl.getDDLStatement());
        }

        // TODO(jackson): Perform some kind of checksuming on the DDL
        // statements so that we notice if the existing tables were created
        // from a different configuration? Or store the configuration
        // itself in Oracle so that they *must* run a program to reconfigure.
    }

    private boolean createTableIfNotExists(
            final Connection conn, final String query) throws SQLException {
        logger.info("Creating table: \n{}", query);
        try (PreparedStatement ps = conn.prepareStatement(query)) {
            ps.executeUpdate();
        } catch(SQLSyntaxErrorException ex) {
            // If "ORA-00955: name is already used by an existing object",
            // the table already exists. Otherwise, it's an unexpected exception.
            if (ex.getErrorCode() != 955) {
                throw ex;
            }
            return false;
        }
        return true;
    }

    /**
     * Processes transactions, reading them from the transaction feed
     * and inserting them into the configured Oracle database. This
     * function blocks indefinitely. Errors are logged to log4j2.
     *
     * TODO(jackson): Support propagating persistent errors?
     */
    public void process() {
        final String insertTxQ = mTransactionsTbl.getInsertStatement();
        final String insertInputQ = mTransactionInputsTbl.getInsertStatement();
        final String insertOutputQ = mTransactionOutputsTbl.getInsertStatement();

        // TODO(jackson): Batch insert transactions instead of inserting
        // them individually. We'll need to circumvent the SDK's transaction
        // feed interface.
        for (;;) {
            try (Connection conn = mDataSource.getConnection()) {
                // Manage our own SQL transactions so we can make this whole
                // blockchain transaction atomic.
                conn.setAutoCommit(false);

                final Transaction tx = mFeed.next(mChain);
                ThreadContext.put("tx_id", tx.id);
                logger.debug("Importing transaction {}", tx.id);

                // Insert the transaction itself.
                try (PreparedStatement ps = conn.prepareStatement(insertTxQ)) {
                    ps.setString(1, tx.id);
                    ps.setLong(2, tx.blockHeight);
                    ps.setTimestamp(3, new Timestamp(tx.timestamp.getTime()));
                    ps.setInt(4, tx.position);
                    ps.setString(5, "yes".equals(tx.isLocal) ? TRUE : FALSE);
                    ps.setBlob(6, asJsonBlob(tx.referenceData));
                    ps.setBlob(7, asJsonBlob(tx));
                    ps.executeUpdate();
                }

                // Insert each of the inputs.
                try (PreparedStatement ps = conn.prepareStatement(insertInputQ)) {
                    for (int i = 0; i < tx.inputs.size(); i++) {
                        final Transaction.Input input = tx.inputs.get(i);

                        ps.setString(1, tx.id);
                        ps.setInt(2, i);
                        ps.setString(3, input.type);
                        ps.setString(4, input.assetId);
                        ps.setString(5, input.assetAlias);
                        ps.setBlob(6, asJsonBlob(input.assetDefinition));
                        ps.setBlob(7, asJsonBlob(input.assetTags));
                        ps.setString(8, "yes".equals(input.assetIsLocal) ? TRUE : FALSE);
                        ps.setLong(9, input.amount);
                        ps.setString(10, input.accountId);
                        ps.setString(11, input.accountAlias);
                        ps.setBlob(12, asJsonBlob(input.accountTags));
                        ps.setString(13, input.issuanceProgram); // clob
                        ps.setBlob(14, asJsonBlob(input.referenceData));
                        ps.setString(15, "yes".equals(input.isLocal) ? TRUE : FALSE);
                        ps.setString(16, input.spentOutputId);

                        // TODO(jackson): We can't use addBatch in Oracle
                        // earlier than 12.1. If customers need support for
                        // Oracle < 12.1, we'll need to use the deprecated
                        // OraclePreparedStatement.setExecuteBatch method:
                        // http://docs.oracle.com/cd/B28359_01/java.111/b31224/oraperf.htm
                        ps.addBatch();
                    }
                    ps.executeBatch();
                }

                // Insert each of the outputs.
                try (PreparedStatement ps = conn.prepareStatement(insertOutputQ)) {
                    for (int i = 0; i < tx.outputs.size(); i++) {
                        final Transaction.Output output = tx.outputs.get(i);

                        ps.setString(1, tx.id);
                        ps.setInt(2, i);
                        ps.setString(3, output.id);
                        ps.setString(4, output.type);
                        ps.setString(5, output.purpose);
                        ps.setString(6, output.assetId);
                        ps.setString(7, output.assetAlias);
                        ps.setBlob(8, asJsonBlob(output.assetDefinition));
                        ps.setBlob(9, asJsonBlob(output.assetTags));
                        ps.setString(10, "yes".equals(output.assetIsLocal) ? TRUE : FALSE);
                        ps.setLong(11, output.amount);
                        ps.setString(12, output.accountId);
                        ps.setString(13, output.accountAlias);
                        ps.setBlob(14, asJsonBlob(output.accountTags));
                        ps.setString(15, output.controlProgram); // clob
                        ps.setBlob(16, asJsonBlob(output.referenceData));
                        ps.setString(17, "yes".equals(output.isLocal) ? TRUE : FALSE);
                        ps.setString(18, FALSE);

                        ps.addBatch();
                    }
                    ps.executeBatch();
                }

                // Commit the entire transaction at once.
                conn.commit();

                // Mark this transaction as processed, now that we've
                // successfully indexed it.
                mFeed.ack(mChain);
            } catch (SQLException ex) {
                // We can hit a unique constraint violation (ORA-00001)
                // iff we already processed this transaction but have not
                // yet successfully acked the transaction. If that's the
                // case, just ack the transaction and move on to the next
                // blockchain transaction.
                if (ex.getErrorCode() == 1) {
                    logger.info("Processed transaction twice; acking and continuing");
                    try {
                        mFeed.ack(mChain);
                    } catch(ChainException chainEx) {
                        logger.catching(chainEx);
                    }
                } else {
                    // If it's a different error code, log the error, don't
                    // ack and try again.
                    logger.catching(ex);
                }
            } catch (ChainException ex) {
                logger.catching(ex);
            } finally {
                ThreadContext.remove("tx_id");
            }
        }
    }

    private static InputStream asJsonBlob(final Object obj) {
        if (obj == null) return null;
        return new ByteArrayInputStream(
                gson
                .toJson(obj)
                .getBytes(StandardCharsets.UTF_8));
    }
}
