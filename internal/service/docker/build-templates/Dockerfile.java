ARG JAVA_VERSION=21
ARG MAVEN_VERSION=3.9.9
ARG TARGET_NAME=app.jar
ARG APP_PORT=8080

FROM maven:${MAVEN_VERSION}-eclipse-temurin-${JAVA_VERSION}-alpine AS builder

WORKDIR /app

COPY pom.xml .
RUN mvn -B dependency:go-offline

COPY src ./src
RUN mvn -B package -DskipTests

FROM eclipse-temurin:${JAVA_VERSION}-jre-alpine

WORKDIR /app

COPY --from=builder /app/target/${TARGET_NAME} /app/app.jar

RUN addgroup -S app && adduser -S app -G app
USER app

EXPOSE ${APP_PORT}

ENTRYPOINT ["java","-jar","/app/app.jar"]
