package com.chain.api;

import com.chain.common.Utils;

import java.util.Date;

import com.google.gson.annotations.SerializedName;

/**
 * Receivers are used to facilitate payments between accounts on different
 * cores. They contain a control program, expiration date, and in the future,
 * other payment-related metadata.
 *
 * This class supersedes the {@link ControlProgram} class. Receivers are
 * typically created under accounts via the {@link Account.ReceiverBuilder} class.
 */
public class Receiver {
  /**
   * Hex-encoded string representation of the control program.
   */
  @SerializedName("control_program")
  public String controlProgram;

  /**
   * The date after which the receiver is no longer valid for receiving
   * payments.
   */
  @SerializedName("expires_at")
  public Date expiresAt;

  /**
   * Serializes the receiver into a form that is safe to transfer over the wire.
   */
  public String serialize() {
    return Utils.serializer.toJson(this);
  }
}
