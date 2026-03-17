plugins {
    id("com.android.application")
}

android {
    namespace = "com.insidegallery.fingerprint"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.insidegallery.fingerprint"
        minSdk = 21
        targetSdk = 34
        versionCode = 1
        versionName = "1.0"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }

    sourceSets {
        getByName("main") {
            java.srcDirs("src/main/java")
        }
    }
}

dependencies {
    implementation(fileTree(mapOf("dir" to "libs", "include" to listOf("*.aar"))))
}
