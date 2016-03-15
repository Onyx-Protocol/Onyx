package chain.qa;

import java.net.*;

import com.chain.Client;

public class TestClient extends Client {
	public TestClient(URL url) throws MalformedURLException {
		super(url);
	}

	// Client.post expects a return type
	private class Project {
		public String id;
		public String name;
	}

	public String createProject(String name) throws Exception {
		Project project = this.post("/projects", "{\"name\":\"" + name + "\"}", Project.class);
		return project.id;
	}
}
