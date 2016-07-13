package com.chain.qa;

import java.net.*;

import com.chain.Client;

/**
 * TestClient provides access to internal api routes
 * not exposed in Client.
 */
public class TestClient extends Client {
	public TestClient(URL url) throws MalformedURLException {
		super(url);
	}

	/**
	 * Project is used as a JSON deserialization object for Client#post
	 */
	private class Project {
		public String id;
	}

	/**
	 * Creates a project and returns its ID as a string.
	 */
	public String createProject(String name)
	throws Exception {
		Project project = this.apiClient.post("/projects", String.format("{\"name\":\"%s\"}", name), Project.class);
		return project.id;
	}
}
