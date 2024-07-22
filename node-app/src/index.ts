import axios from "axios";
import "./instrumentation";
import express, { Request, Response } from "express";

const app = express();
const port = 3000;

app.get("/test", async (req: Request, res: Response) => {
  try {
    const resp = await axios.get("http://localhost:8080/ping");
    res.send(`Hello World! ${resp.data.message}`);
  } catch (err) {
    res.status(500).json({
      message: err
    })
  }
});

app.listen(port, () => {
  console.log(`Example app listening on port ${port}`);
});
